package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
)

// RefundResult is the outcome of a Refund call.
type RefundResult struct {
	Success  bool
	RefundID string // the payment ID after refund
}

// RefundUsecase handles 100% order refunds back to the customer wallet.
type RefundUsecase interface {
	// Refund credits amount back to customer wallet. Idempotent by order_id.
	Refund(ctx context.Context, orderID string, amount int64) (*RefundResult, error)
}

type refundUsecase struct {
	db          *gorm.DB
	walletRepo  repository.WalletRepository
	ledgerRepo  repository.LedgerRepository
	paymentRepo repository.PaymentRepository
	outboxRepo  repository.OutboxRepository
}

func NewRefundUsecase(
	db *gorm.DB,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	paymentRepo repository.PaymentRepository,
	outboxRepo repository.OutboxRepository,
) RefundUsecase {
	return &refundUsecase{
		db:          db,
		walletRepo:  walletRepo,
		ledgerRepo:  ledgerRepo,
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
	}
}

func (uc *refundUsecase) Refund(ctx context.Context, orderID string, amount int64) (*RefundResult, error) {
	// Idempotency: already refunded is a no-op.
	existing, err := uc.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPaymentNotFound
	}
	if existing.Status == entity.PaymentRefunded {
		return &RefundResult{Success: true, RefundID: existing.ID}, nil
	}

	traceID := outbox.TraceIDFromCtx(ctx)

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Double-entry: debit SYSTEM, credit CUSTOMER.
		if err := doubleEntryTransfer(ctx, transferParams{
			tx:              tx,
			walletRepo:      uc.walletRepo,
			ledgerRepo:      uc.ledgerRepo,
			debitOwnerType:  entity.WalletOwnerSystem,
			debitOwnerID:    entity.SystemOwnerID,
			creditOwnerType: entity.WalletOwnerCustomer,
			creditOwnerID:   existing.CustomerID,
			entryType:       entity.LedgerRefund,
			refType:         entity.LedgerRefOrder,
			refID:           orderID,
			amount:          amount,
			traceID:         traceID,
		}); err != nil {
			return err
		}

		if err := uc.paymentRepo.UpdateStatus(ctx, tx, existing.ID, entity.PaymentRefunded, nil); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"payment_id":  existing.ID,
			"order_id":    orderID,
			"customer_id": existing.CustomerID,
			"amount":      amount,
		})
		// wallet.refunded feeds the customer-facing wallet notification;
		// payment.refunded is the source of truth the order service consumes
		// to flip its payment_status (no dual-write at cancel time).
		if err := uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "wallet",
			AggregateID:   existing.CustomerID,
			EventType:     "wallet.refunded",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		}); err != nil {
			return err
		}
		return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "payment",
			AggregateID:   existing.ID,
			EventType:     "payment.refunded",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		})
	})

	if txErr != nil {
		slog.ErrorContext(ctx, "refund: tx failed", "order_id", orderID, "err", txErr)
		return nil, txErr
	}

	slog.InfoContext(ctx, "refund: credited customer wallet", "order_id", orderID, "amount", amount)
	return &RefundResult{Success: true, RefundID: existing.ID}, nil
}
