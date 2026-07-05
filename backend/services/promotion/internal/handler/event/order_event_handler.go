package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/promotion/internal/repository"
	"project/services/promotion/internal/usecase"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// OrderEventHandler consumes order.events for the promotion service.
// Currently handles order.cancelled → ReleaseUsage (saga compensation).
// Deduped via processed_events committed atomically with the release.
type OrderEventHandler struct {
	db            *gorm.DB
	applyUC       usecase.ApplyUsecase
	processedRepo repository.ProcessedEventRepository
}

func NewOrderEventHandler(db *gorm.DB, applyUC usecase.ApplyUsecase, processedRepo repository.ProcessedEventRepository) *OrderEventHandler {
	return &OrderEventHandler{db: db, applyUC: applyUC, processedRepo: processedRepo}
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

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			slog.DebugContext(ctx, "promotion: order.cancelled already processed", "event_id", env.EventID)
			return nil
		}
		return h.applyUC.ReleaseUsageInTx(ctx, tx, data.OrderID)
	})
	if err != nil {
		slog.ErrorContext(ctx, "promotion: order.cancelled: release usage failed",
			"order_id", data.OrderID, "err", err)
		return err // retryable
	}
	return nil
}
