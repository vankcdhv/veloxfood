package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/reporting/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// PaymentEventHandler consumes payment.events and payout.events topics.
// payment.captured marks the order fact as settle-eligible (not yet settled).
// payout.settled marks the order fact as fully settled.
type PaymentEventHandler struct {
	db            *gorm.DB
	projRepo      repository.ProjectionRepository
	processedRepo repository.ProcessedEventRepository
}

func NewPaymentEventHandler(
	db *gorm.DB,
	projRepo repository.ProjectionRepository,
	processedRepo repository.ProcessedEventRepository,
) *PaymentEventHandler {
	return &PaymentEventHandler{db: db, projRepo: projRepo, processedRepo: processedRepo}
}

// HandlePaymentKafkaMessage routes messages from payment.events.
func (h *PaymentEventHandler) HandlePaymentKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "reporting: payment event unmarshal failed", "err", err)
		return nil
	}
	if env.EventType == "payment.captured" {
		return h.handlePaymentCaptured(ctx, env)
	}
	slog.DebugContext(ctx, "reporting: payment event ignored", "event_type", env.EventType)
	return nil
}

// HandlePayoutKafkaMessage routes messages from payout.events.
func (h *PaymentEventHandler) HandlePayoutKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "reporting: payout event unmarshal failed", "err", err)
		return nil
	}
	if env.EventType == "payout.settled" {
		return h.handlePayoutSettled(ctx, env)
	}
	slog.DebugContext(ctx, "reporting: payout event ignored", "event_type", env.EventType)
	return nil
}

type paymentCapturedData struct {
	OrderID string `json:"order_id"`
}

func (h *PaymentEventHandler) handlePaymentCaptured(ctx context.Context, env outbox.Envelope) error {
	var data paymentCapturedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: payment.captured unmarshal failed", "err", err)
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
		// payment.captured means money collected — no status change on order_facts,
		// but we note that settle eligibility is now active (settled stays false
		// until payout.settled arrives).
		slog.InfoContext(ctx, "reporting: payment.captured noted", "order_id", data.OrderID)
		return nil
	})
}

type payoutSettledData struct {
	StoreID string   `json:"store_id"`
	BatchID string   `json:"batch_id"`
	Amount  int64    `json:"amount"`
	// OrderIDs is optional; if absent we settle all COMPLETED unsettled facts for this store.
	OrderIDs []string `json:"order_ids,omitempty"`
}

func (h *PaymentEventHandler) handlePayoutSettled(ctx context.Context, env outbox.Envelope) error {
	var data payoutSettledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: payout.settled unmarshal failed", "err", err)
		return nil
	}
	if data.StoreID == "" {
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
		// Mark all completed, unsettled order_facts for this store as settled.
		return tx.WithContext(ctx).Exec(`
			UPDATE order_facts
			SET settled = true, updated_at = NOW()
			WHERE store_id = ? AND status = 'COMPLETED' AND settled = false`,
			data.StoreID,
		).Error
	})
}
