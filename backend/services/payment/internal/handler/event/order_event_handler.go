package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"
	"project/services/payment/internal/usecase"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// OrderEventHandler consumes order.events Kafka topic.
// Handlers are idempotent via processed_events table.
type OrderEventHandler struct {
	db                   *gorm.DB
	paymentRepo          repository.PaymentRepository
	walletRepo           repository.WalletRepository
	ledgerRepo           repository.LedgerRepository
	outboxRepo           repository.OutboxRepository
	processedEventRepo   repository.ProcessedEventRepository
	refundUC             usecase.RefundUsecase
}

func NewOrderEventHandler(
	db *gorm.DB,
	paymentRepo repository.PaymentRepository,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	outboxRepo repository.OutboxRepository,
	processedEventRepo repository.ProcessedEventRepository,
	refundUC usecase.RefundUsecase,
) *OrderEventHandler {
	return &OrderEventHandler{
		db:                 db,
		paymentRepo:        paymentRepo,
		walletRepo:         walletRepo,
		ledgerRepo:         ledgerRepo,
		outboxRepo:         outboxRepo,
		processedEventRepo: processedEventRepo,
		refundUC:           refundUC,
	}
}

// HandleKafkaMessage routes messages from order.events to per-type handlers.
func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "order event received", "topic", msg.Topic, "key", string(msg.Key))

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "order event: unmarshal envelope failed", "err", err)
		return nil // non-retryable parse error — commit offset
	}

	switch env.EventType {
	case "order.placed":
		return h.handleOrderPlaced(ctx, env)
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
	default:
		slog.DebugContext(ctx, "order event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

// orderPlacedData matches the payload published by the Order service.
type orderPlacedData struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Method     string `json:"method"` // COD | MOMO | WALLET
	Amount     int64  `json:"amount"`
}

// handleOrderPlaced records a payment intent for the order.
// WALLET/MOMO: upsert PENDING record if absent (saga already called Capture for WALLET).
// COD: create Payment COD PENDING (no ledger movement — settled on delivery).
func (h *OrderEventHandler) handleOrderPlaced(ctx context.Context, env outbox.Envelope) error {
	var data orderPlacedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order.placed: unmarshal failed", "err", err)
		return nil
	}
	if data.OrderID == "" || data.CustomerID == "" {
		return nil
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency via processed_events.
		inserted, err := h.processedEventRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			slog.DebugContext(ctx, "order.placed: already processed", "event_id", env.EventID)
			return nil
		}

		// Check if payment already recorded (Capture was called by saga).
		existing, err := h.paymentRepo.GetByOrderID(ctx, data.OrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil // already recorded by Capture call
		}

		// Only COD needs a record here; WALLET/MOMO already captured via gRPC.
		if entity.PaymentMethod(data.Method) != entity.MethodCOD {
			return nil
		}

		p := &entity.Payment{
			OrderID:    &data.OrderID,
			CustomerID: data.CustomerID,
			Method:     entity.MethodCOD,
			Provider:   entity.ProviderInternal,
			Env:        "demo",
			Amount:     data.Amount,
			Status:     entity.PaymentPending,
		}
		return h.paymentRepo.Create(ctx, tx, p)
	})
}

// orderCancelledData matches the payload published by the Order service.
type orderCancelledData struct {
	OrderID        string `json:"order_id"`
	WasPaidOnline  bool   `json:"was_paid_online"`
	Amount         int64  `json:"amount"`
}

// handleOrderCancelled refunds 100% to the customer wallet when the order was
// paid online (WALLET or MOMO/captured). COD cancellations require no ledger action.
func (h *OrderEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	var data orderCancelledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order.cancelled: unmarshal failed", "err", err)
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

		if !data.WasPaidOnline {
			return nil
		}

		// Delegate to refund usecase (idempotent by order_id).
		_, refErr := h.refundUC.Refund(ctx, data.OrderID, data.Amount)
		if refErr != nil {
			slog.ErrorContext(ctx, "order.cancelled: refund failed", "order_id", data.OrderID, "err", refErr)
			return refErr
		}
		return nil
	})
}
