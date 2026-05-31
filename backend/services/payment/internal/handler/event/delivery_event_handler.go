package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// DeliveryEventHandler consumes delivery.events Kafka topic.
// Handles COD order delivery → credits STORE_PAYABLE wallet.
type DeliveryEventHandler struct {
	db                 *gorm.DB
	paymentRepo        repository.PaymentRepository
	walletRepo         repository.WalletRepository
	ledgerRepo         repository.LedgerRepository
	outboxRepo         repository.OutboxRepository
	processedEventRepo repository.ProcessedEventRepository
}

func NewDeliveryEventHandler(
	db *gorm.DB,
	paymentRepo repository.PaymentRepository,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	outboxRepo repository.OutboxRepository,
	processedEventRepo repository.ProcessedEventRepository,
) *DeliveryEventHandler {
	return &DeliveryEventHandler{
		db:                 db,
		paymentRepo:        paymentRepo,
		walletRepo:         walletRepo,
		ledgerRepo:         ledgerRepo,
		outboxRepo:         outboxRepo,
		processedEventRepo: processedEventRepo,
	}
}

// HandleKafkaMessage routes messages from delivery.events to per-type handlers.
func (h *DeliveryEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "delivery event received", "topic", msg.Topic, "key", string(msg.Key))

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "delivery event: unmarshal envelope failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "order.delivered":
		return h.handleOrderDelivered(ctx, env)
	default:
		slog.DebugContext(ctx, "delivery event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

// orderDeliveredData matches the payload published by the Delivery/Order service.
type orderDeliveredData struct {
	OrderID string `json:"order_id"`
	StoreID string `json:"store_id"`
	Amount  int64  `json:"amount"`
}

// handleOrderDelivered marks a COD payment as CAPTURED and credits STORE_PAYABLE.
// Idempotent via processed_events.
func (h *DeliveryEventHandler) handleOrderDelivered(ctx context.Context, env outbox.Envelope) error {
	var data orderDeliveredData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order.delivered: unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" || data.StoreID == "" {
		return nil
	}

	traceID := env.TraceID

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedEventRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			slog.DebugContext(ctx, "order.delivered: already processed", "event_id", env.EventID)
			return nil
		}

		payment, err := h.paymentRepo.GetByOrderID(ctx, data.OrderID)
		if err != nil {
			return err
		}
		if payment == nil {
			slog.WarnContext(ctx, "order.delivered: no payment found", "order_id", data.OrderID)
			return nil
		}
		// Only act on COD orders; online payments are settled at capture time.
		if payment.Method != entity.MethodCOD {
			return nil
		}
		// Idempotent: already captured means store was already credited.
		if payment.Status == entity.PaymentCaptured {
			return nil
		}

		amount := data.Amount
		if amount == 0 {
			amount = payment.Amount
		}

		tracePtr := &traceID
		if traceID == "" {
			tracePtr = nil
		}

		// Double-entry: debit SYSTEM, credit STORE_PAYABLE (COD cash collected).
		debitWallet, err := h.walletRepo.GetOrCreateForUpdate(ctx, tx, entity.WalletOwnerSystem, entity.SystemOwnerID)
		if err != nil {
			return err
		}
		creditWallet, err := h.walletRepo.GetOrCreateForUpdate(ctx, tx, entity.WalletOwnerStorePayable, data.StoreID)
		if err != nil {
			return err
		}

		newDebit := debitWallet.Balance - amount
		newCredit := creditWallet.Balance + amount

		if err := h.ledgerRepo.Append(ctx, tx, &entity.LedgerEntry{
			WalletID:     debitWallet.ID,
			EntryType:    entity.LedgerPayment,
			Amount:       -amount,
			RefType:      entity.LedgerRefOrder,
			RefID:        data.OrderID,
			BalanceAfter: newDebit,
			TraceID:      tracePtr,
		}); err != nil {
			return err
		}
		if err := h.ledgerRepo.Append(ctx, tx, &entity.LedgerEntry{
			WalletID:     creditWallet.ID,
			EntryType:    entity.LedgerPayment,
			Amount:       amount,
			RefType:      entity.LedgerRefOrder,
			RefID:        data.OrderID,
			BalanceAfter: newCredit,
			TraceID:      tracePtr,
		}); err != nil {
			return err
		}
		if err := h.walletRepo.UpdateBalance(ctx, tx, debitWallet.ID, newDebit); err != nil {
			return err
		}
		if err := h.walletRepo.UpdateBalance(ctx, tx, creditWallet.ID, newCredit); err != nil {
			return err
		}

		if err := h.paymentRepo.UpdateStatus(ctx, tx, payment.ID, entity.PaymentCaptured, nil); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"payment_id": payment.ID,
			"order_id":   data.OrderID,
			"store_id":   data.StoreID,
			"amount":     amount,
			"method":     string(entity.MethodCOD),
		})
		return h.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "payment",
			AggregateID:   payment.ID,
			EventType:     "payment.captured",
			Payload:       payload,
			TraceID:       tracePtr,
		})
	})
}
