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
	// Refund credits amount back to the customer wallet in its own transaction.
	// Idempotent by order_id; a missing payment row is a null compensation and
	// succeeds as a no-op.
	Refund(ctx context.Context, orderID string, amount int64) (*RefundResult, error)

	// RefundInTx runs the same refund inside a caller-owned transaction (DTM branch
	// handlers run it under the sub-transaction barrier).
	RefundInTx(ctx context.Context, tx *gorm.DB, orderID string, amount int64) (*RefundResult, error)
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

// Refund opens its own transaction and defers to RefundInTx, so both entry points
// share one refund semantics. Every caller of this path is a compensation that
// cannot know whether the capture ran: the order-service compensation worker and
// the inline-saga rollback both fire on an unknown outcome, and the
// order.cancelled consumer also fires for COD orders that never captured. A
// missing payment row is therefore a null compensation, not an anomaly —
// returning an error here made the compensation worker retry a refund that could
// never succeed until it burned its attempt budget.
func (uc *refundUsecase) Refund(ctx context.Context, orderID string, amount int64) (*RefundResult, error) {
	var result *RefundResult
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r, err := uc.RefundInTx(ctx, tx, orderID, amount)
		if err != nil {
			return err
		}
		result = r
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "refund: tx failed", "order_id", orderID, "err", txErr)
		return nil, txErr
	}
	return result, nil
}

// RefundInTx is the DTM-branch variant: it joins the caller's transaction and
// treats "capture never happened" as a successful no-op (null compensation).
func (uc *refundUsecase) RefundInTx(ctx context.Context, tx *gorm.DB, orderID string, amount int64) (*RefundResult, error) {
	existing, err := uc.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		slog.WarnContext(ctx, "refund: no payment row — null compensation, skipping", "order_id", orderID)
		return &RefundResult{Success: true}, nil
	}
	if r := refundShortCircuit(ctx, existing, orderID); r != nil {
		return r, nil
	}
	if err := uc.refundLocked(ctx, tx, existing, orderID, amount); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "refund: credited customer wallet", "order_id", orderID, "amount", amount)
	return &RefundResult{Success: true, RefundID: existing.ID}, nil
}

// refundShortCircuit resolves the idempotent/no-op cases: already refunded, or
// a payment that never reached CAPTURED (PENDING MoMo intent, FAILED attempt,
// COD) — no money was taken, so crediting the wallet would mint free money.
func refundShortCircuit(ctx context.Context, existing *entity.Payment, orderID string) *RefundResult {
	if existing.Status == entity.PaymentRefunded {
		return &RefundResult{Success: true, RefundID: existing.ID}
	}
	if existing.Status != entity.PaymentCaptured {
		slog.WarnContext(ctx, "refund: payment never captured — skipping credit",
			"order_id", orderID, "status", existing.Status)
		return &RefundResult{Success: true, RefundID: existing.ID}
	}
	return nil
}

// refundLocked performs the double-entry credit, status flip and refund events
// inside the given transaction.
func (uc *refundUsecase) refundLocked(ctx context.Context, tx *gorm.DB, existing *entity.Payment, orderID string, amount int64) error {
	traceID := outbox.TraceIDFromCtx(ctx)

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
}
