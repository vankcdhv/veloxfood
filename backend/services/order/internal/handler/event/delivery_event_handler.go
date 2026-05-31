package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/order/internal/entity"
	"project/services/order/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// DeliveryEventHandler consumes delivery.events for the order service.
// delivery.claimed         → SHIPPER_ASSIGNED
// delivery.status_changed  → sync DELIVERING/DELIVERED; publish order.delivered when status=DELIVERED
type DeliveryEventHandler struct {
	db                 *gorm.DB
	orderRepo          repository.OrderRepository
	outboxRepo         repository.OutboxRepository
	processedEventRepo repository.ProcessedEventRepository
}

func NewDeliveryEventHandler(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	outboxRepo repository.OutboxRepository,
	processedEventRepo repository.ProcessedEventRepository,
) *DeliveryEventHandler {
	return &DeliveryEventHandler{
		db:                 db,
		orderRepo:          orderRepo,
		outboxRepo:         outboxRepo,
		processedEventRepo: processedEventRepo,
	}
}

func (h *DeliveryEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "order: delivery event received", "topic", msg.Topic)

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "order: delivery event: unmarshal envelope failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "delivery.claimed":
		return h.handleDeliveryClaimed(ctx, env)
	case "delivery.status_changed":
		return h.handleDeliveryStatusChanged(ctx, env)
	default:
		slog.DebugContext(ctx, "order: delivery event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

// deliveryClaimedData matches the frozen delivery.claimed payload (§2bis).
type deliveryClaimedData struct {
	OrderID   string `json:"order_id"`
	ShipperID string `json:"shipper_id"`
	BatchID   string `json:"batch_id"`
}

func (h *DeliveryEventHandler) handleDeliveryClaimed(ctx context.Context, env outbox.Envelope) error {
	var data deliveryClaimedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order: delivery.claimed: unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" {
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedEventRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return h.orderRepo.UpdateStatus(ctx, tx, data.OrderID, entity.StatusShipperAssigned,
			strPtr(data.ShipperID), strPtr("shipper claimed"))
	})
}

// deliveryStatusChangedData matches the frozen delivery.status_changed payload (§2bis).
type deliveryStatusChangedData struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"` // PICKED_UP | DELIVERING | DELIVERED
	ShipperID string `json:"shipper_id"`
}

func (h *DeliveryEventHandler) handleDeliveryStatusChanged(ctx context.Context, env outbox.Envelope) error {
	var data deliveryStatusChangedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order: delivery.status_changed: unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" {
		return nil
	}

	var nextStatus entity.OrderStatus
	switch data.Status {
	case "DELIVERING":
		nextStatus = entity.StatusDelivering
	case "DELIVERED":
		nextStatus = entity.StatusDelivered
	default:
		// PICKED_UP and other statuses are informational for order — no transition.
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedEventRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}

		order, err := h.orderRepo.GetByIDForUpdate(ctx, tx, data.OrderID)
		if err != nil {
			slog.WarnContext(ctx, "order: delivery.status_changed: order not found", "order_id", data.OrderID)
			return nil // non-retryable — order may not exist yet in edge cases
		}

		if !entity.CanTransition(order.Status, nextStatus) {
			slog.WarnContext(ctx, "order: delivery.status_changed: invalid transition",
				"from", order.Status, "to", nextStatus)
			return nil
		}

		if err := h.orderRepo.UpdateStatus(ctx, tx, data.OrderID, nextStatus,
			strPtr(data.ShipperID), nil); err != nil {
			return err
		}

		// When delivered: publish order.delivered (frozen payload) + order.completed.
		if nextStatus == entity.StatusDelivered {
			deliveredPayload, _ := json.Marshal(map[string]any{
				"order_id":       order.ID,
				"store_id":       order.StoreID,
				"amount":         order.GrandTotal,
				"payment_method": string(order.PaymentMethod),
			})
			if err := h.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
				AggregateType: "order",
				AggregateID:   order.ID,
				EventType:     "order.delivered",
				Payload:       deliveredPayload,
				TraceID:       strPtr(env.TraceID),
			}); err != nil {
				return err
			}

			// MVP: publish order.completed immediately after delivered.
			completedPayload, _ := json.Marshal(map[string]any{"order_id": order.ID})
			return h.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
				AggregateType: "order",
				AggregateID:   order.ID,
				EventType:     "order.completed",
				Payload:       completedPayload,
				TraceID:       strPtr(env.TraceID),
			})
		}
		return nil
	})
}

// strPtr is a package-level helper shared across event handlers.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
