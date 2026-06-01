package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"

	"github.com/segmentio/kafka-go"
)

// OrderEventHandler consumes order.events for the review service.
// order.completed → log eligibility (actual review creation is customer-initiated via HTTP).
// Idempotency is ensured by processed_events; this handler is intentionally lightweight.
type OrderEventHandler struct{}

func NewOrderEventHandler() *OrderEventHandler { return &OrderEventHandler{} }

func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "review: order event unmarshal failed", "err", err)
		return nil // non-retryable parse error
	}
	switch env.EventType {
	case "order.completed":
		var data struct {
			OrderID string `json:"order_id"`
		}
		if err := json.Unmarshal(env.Data, &data); err == nil {
			slog.InfoContext(ctx, "review: order completed — review eligible",
				"order_id", data.OrderID, "event_id", env.EventID)
		}
	default:
		slog.DebugContext(ctx, "review: order event unhandled", "event_type", env.EventType)
	}
	return nil
}
