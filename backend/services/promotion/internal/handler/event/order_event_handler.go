package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/promotion/internal/usecase"

	"github.com/segmentio/kafka-go"
)

// OrderEventHandler consumes order.events for the promotion service.
// Currently handles order.cancelled → ReleaseUsage (idempotent, saga compensation).
type OrderEventHandler struct {
	applyUC usecase.ApplyUsecase
}

func NewOrderEventHandler(applyUC usecase.ApplyUsecase) *OrderEventHandler {
	return &OrderEventHandler{applyUC: applyUC}
}

// HandleKafkaMessage routes order.events messages to the correct handler.
func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "promotion: order event received", "topic", msg.Topic, "key", string(msg.Key))

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "promotion: order event: unmarshal envelope failed", "err", err)
		return nil // non-retryable parse error — commit offset
	}

	switch env.EventType {
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
	default:
		slog.DebugContext(ctx, "promotion: order event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

type orderCancelledData struct {
	OrderID string `json:"order_id"`
}

// handleOrderCancelled releases any RESERVED promotion usages for the order.
// Idempotent: ReleaseUsage is a no-op when no RESERVED rows exist.
func (h *OrderEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	var data orderCancelledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "promotion: order.cancelled: unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" {
		return nil
	}

	if err := h.applyUC.ReleaseUsage(ctx, data.OrderID); err != nil {
		slog.ErrorContext(ctx, "promotion: order.cancelled: release usage failed",
			"order_id", data.OrderID, "err", err)
		return err // retryable
	}
	return nil
}
