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

// PaymentEventHandler consumes payment.events for the order service.
// Handles: payment.captured → mark order PAID; payment.refunded → mark order
// REFUNDED (payment service is the source of truth — the order no longer sets
// this synchronously at cancel time); payment.failed → log.
type PaymentEventHandler struct {
	db                 *gorm.DB
	orderRepo          repository.OrderRepository
	processedEventRepo repository.ProcessedEventRepository
}

func NewPaymentEventHandler(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	processedEventRepo repository.ProcessedEventRepository,
) *PaymentEventHandler {
	return &PaymentEventHandler{db: db, orderRepo: orderRepo, processedEventRepo: processedEventRepo}
}

func (h *PaymentEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "order: payment event received", "topic", msg.Topic)

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "order: payment event: unmarshal envelope failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "payment.captured":
		return h.handlePaymentCaptured(ctx, env)
	case "payment.refunded":
		return h.handlePaymentRefunded(ctx, env)
	case "payment.failed":
		return h.handlePaymentFailed(ctx, env)
	default:
		slog.DebugContext(ctx, "order: payment event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

type paymentCapturedData struct {
	OrderID string `json:"order_id"`
}

func (h *PaymentEventHandler) handlePaymentCaptured(ctx context.Context, env outbox.Envelope) error {
	var data paymentCapturedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order: payment.captured: unmarshal failed", "err", err)
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
		return h.orderRepo.UpdatePaymentStatus(ctx, tx, data.OrderID, entity.PaymentPaid)
	})
}

// handlePaymentRefunded flips the order's payment_status once the refund has
// actually been credited by the payment service.
func (h *PaymentEventHandler) handlePaymentRefunded(ctx context.Context, env outbox.Envelope) error {
	var data paymentCapturedData // same shape: {order_id}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order: payment.refunded: unmarshal failed", "err", err)
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
		return h.orderRepo.UpdatePaymentStatus(ctx, tx, data.OrderID, entity.PaymentRefunded)
	})
}

type paymentFailedData struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

func (h *PaymentEventHandler) handlePaymentFailed(ctx context.Context, env outbox.Envelope) error {
	var data paymentFailedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil
	}
	slog.WarnContext(ctx, "order: payment failed", "order_id", data.OrderID, "reason", data.Reason)
	return nil
}
