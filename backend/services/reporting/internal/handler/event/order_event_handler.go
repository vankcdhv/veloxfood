package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/pkg/outbox"
	"project/services/reporting/internal/entity"
	"project/services/reporting/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// OrderEventHandler consumes order.events and updates order_facts + revenue_daily.
type OrderEventHandler struct {
	db            *gorm.DB
	projRepo      repository.ProjectionRepository
	processedRepo repository.ProcessedEventRepository
}

func NewOrderEventHandler(
	db *gorm.DB,
	projRepo repository.ProjectionRepository,
	processedRepo repository.ProcessedEventRepository,
) *OrderEventHandler {
	return &OrderEventHandler{db: db, projRepo: projRepo, processedRepo: processedRepo}
}

// HandleKafkaMessage routes order.events messages to per-type handlers.
func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "reporting: order event unmarshal failed", "err", err)
		return nil // non-retryable — commit offset
	}

	switch env.EventType {
	case "order.placed":
		return h.handleOrderPlaced(ctx, env)
	case "order.completed":
		return h.handleOrderStatusChange(ctx, env, "COMPLETED")
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
	default:
		slog.DebugContext(ctx, "reporting: order event ignored", "event_type", env.EventType)
	}
	return nil
}

// orderPlacedData matches the frozen order.placed payload (02-domain-events §2bis).
type orderPlacedData struct {
	OrderID       string `json:"order_id"`
	Code          string `json:"code"`
	CustomerID    string `json:"customer_id"`
	StoreID       string `json:"store_id"`
	Fulfillment   string `json:"fulfillment"`
	PaymentMethod string `json:"payment_method"`
	ItemsTotal    int64  `json:"items_total"`
	ShipFee       int64  `json:"ship_fee"`
	Discount      int64  `json:"discount"`
	GrandTotal    int64  `json:"grand_total"`
	Items         []struct {
		Date string `json:"date"` // YYYY-MM-DD — use first item's date as the fact date
	} `json:"items"`
}

func (h *OrderEventHandler) handleOrderPlaced(ctx context.Context, env outbox.Envelope) error {
	var data orderPlacedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: order.placed unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" || data.StoreID == "" {
		return nil
	}

	// Determine the order date: use the first item's slot date if present,
	// else today's UTC date.
	orderDate := time.Now().UTC().Truncate(24 * time.Hour)
	if len(data.Items) > 0 && data.Items[0].Date != "" {
		if d, err := time.Parse("2006-01-02", data.Items[0].Date); err == nil {
			orderDate = d
		}
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			slog.DebugContext(ctx, "reporting: order.placed already processed", "event_id", env.EventID)
			return nil
		}

		fact := &entity.OrderFact{
			OrderID:       data.OrderID,
			Code:          data.Code,
			StoreID:       data.StoreID,
			CustomerID:    data.CustomerID,
			Date:          orderDate,
			Fulfillment:   data.Fulfillment,
			ItemsTotal:    data.ItemsTotal,
			ShipFee:       data.ShipFee,
			Discount:      data.Discount,
			GrandTotal:    data.GrandTotal,
			PaymentMethod: data.PaymentMethod,
			Status:        "PENDING",
		}
		if err := h.projRepo.UpsertOrderFact(ctx, tx, fact); err != nil {
			return err
		}

		// Increment revenue_daily with the food + ship amounts for this order.
		return h.projRepo.IncrementRevenueDaily(ctx, tx, data.StoreID, orderDate,
			data.ItemsTotal, data.ShipFee, 1, 0)
	})
}

// orderSimpleData covers order.completed and order.cancelled (minimal payload).
type orderSimpleData struct {
	OrderID string `json:"order_id"`
}

type orderCancelledData struct {
	OrderID  string `json:"order_id"`
	StoreID  string `json:"store_id"`
	CustomerID string `json:"customer_id"`
}

func (h *OrderEventHandler) handleOrderStatusChange(ctx context.Context, env outbox.Envelope, newStatus string) error {
	var data orderSimpleData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: order status event unmarshal failed", "event_type", env.EventType, "err", err)
		return nil
	}
	if data.OrderID == "" {
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return h.projRepo.UpdateOrderFactStatus(ctx, tx, data.OrderID, newStatus)
	})
}

func (h *OrderEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	var data orderCancelledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: order.cancelled unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" {
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		if err := h.projRepo.UpdateOrderFactStatus(ctx, tx, data.OrderID, "CANCELLED"); err != nil {
			return err
		}
		// Increment cancelled_count for the store on today's date.
		// We don't know store_id/date from order.cancelled payload alone unless the
		// fact row exists — use a direct SQL update on revenue_daily via a sub-query.
		return tx.WithContext(ctx).Exec(`
			UPDATE revenue_daily rd
			SET cancelled_count = rd.cancelled_count + 1
			FROM order_facts of
			WHERE rd.store_id = of.store_id
			  AND rd.date     = of.date
			  AND of.order_id = ?`, data.OrderID,
		).Error
	})
}
