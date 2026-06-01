package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// OrderEventHandler consumes order.events for the delivery service.
//
//   - order.ready          → create AVAILABLE delivery (DELIVERY fulfillment only)
//   - order.cutoff_reached → move delivery to STORE_DELIVERING (remove from available pool)
//   - order.cancelled      → cancel delivery
type OrderEventHandler struct {
	db             *gorm.DB
	deliveryRepo   repository.DeliveryRepository
	processedRepo  repository.ProcessedEventRepository
}

func NewOrderEventHandler(
	db *gorm.DB,
	deliveryRepo repository.DeliveryRepository,
	processedRepo repository.ProcessedEventRepository,
) *OrderEventHandler {
	return &OrderEventHandler{
		db:            db,
		deliveryRepo:  deliveryRepo,
		processedRepo: processedRepo,
	}
}

func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "delivery: order event received", "topic", msg.Topic)

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "delivery: order event: unmarshal envelope failed", "err", err)
		return nil // non-retryable — bad message format
	}

	switch env.EventType {
	case "order.ready":
		return h.handleOrderReady(ctx, env)
	case "order.cutoff_reached":
		return h.handleCutoffReached(ctx, env)
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
	default:
		slog.DebugContext(ctx, "delivery: order event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

// orderReadyData is the enriched order.ready payload (§2bis frozen contract).
type orderReadyData struct {
	OrderID     string `json:"order_id"`
	Code        string `json:"code"`
	Status      string `json:"status"`
	StoreID     string `json:"store_id"`
	LocationID  string `json:"location_id"`
	ShipFee     int64  `json:"ship_fee"`
	Fulfillment string `json:"fulfillment"`
	CustomerID  string `json:"customer_id"`
}

func (h *OrderEventHandler) handleOrderReady(ctx context.Context, env outbox.Envelope) error {
	var data orderReadyData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "delivery: order.ready: unmarshal failed", "err", err)
		return nil
	}

	// Only DELIVERY fulfillment gets a delivery record; PICKUP skips.
	if data.Fulfillment != "DELIVERY" {
		slog.DebugContext(ctx, "delivery: order.ready: skipping non-DELIVERY fulfillment",
			"order_id", data.OrderID, "fulfillment", data.Fulfillment)
		return nil
	}
	if data.OrderID == "" || data.StoreID == "" {
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			slog.DebugContext(ctx, "delivery: order.ready: already processed", "event_id", env.EventID)
			return nil // idempotent
		}

		d := &entity.Delivery{
			OrderID:    data.OrderID,
			OrderCode:  data.Code,
			StoreID:    data.StoreID,
			LocationID: data.LocationID,
			CustomerID: data.CustomerID,
			ShipFee:    data.ShipFee,
			Status:     entity.DeliveryAvailable,
		}
		if err := h.deliveryRepo.Create(ctx, tx, d); err != nil {
			slog.ErrorContext(ctx, "delivery: order.ready: create delivery failed",
				"order_id", data.OrderID, "err", err)
			return err
		}
		slog.InfoContext(ctx, "delivery: AVAILABLE delivery created", "order_id", data.OrderID)
		return nil
	})
}

// cutoffReachedData is the order.cutoff_reached payload.
type cutoffReachedData struct {
	OrderID string `json:"order_id"`
	StoreID string `json:"store_id"`
}

func (h *OrderEventHandler) handleCutoffReached(ctx context.Context, env outbox.Envelope) error {
	var data cutoffReachedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "delivery: order.cutoff_reached: unmarshal failed", "err", err)
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
		// Move AVAILABLE delivery to STORE_DELIVERING — store handles it themselves.
		return h.deliveryRepo.SetStoreDelivering(ctx, tx, data.OrderID)
	})
}

// orderCancelledData is the order.cancelled payload (partial — only order_id needed here).
type orderCancelledData struct {
	OrderID string `json:"order_id"`
}

func (h *OrderEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	var data orderCancelledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "delivery: order.cancelled: unmarshal failed", "err", err)
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
		return h.deliveryRepo.Cancel(ctx, tx, data.OrderID)
	})
}
