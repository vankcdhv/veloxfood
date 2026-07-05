package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"project/pkg/outbox"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/infrastructure/momo"
	"project/services/payment/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaptureRequest is the saga input from the order service.
type CaptureRequest struct {
	OrderID    string
	CustomerID string
	Amount     int64
	Method     entity.PaymentMethod
}

// CaptureResult is the outcome of a Capture call.
type CaptureResult struct {
	PaymentID string
	Status    entity.PaymentStatus
	PayURL    string // non-empty for MOMO
	Error     string // non-empty on failure
}

// CaptureUsecase handles payment capture for the order saga.
type CaptureUsecase interface {
	Capture(ctx context.Context, req CaptureRequest) (*CaptureResult, error)

	// CaptureInTx performs the database part of a capture inside a caller-owned
	// transaction (DTM branch handlers run it under the barrier). WALLET is a
	// full synchronous capture; MOMO/COD only create the PENDING intent row.
	// When the result is a MoMo intent without a pay_url the caller must invoke
	// FinishMoMoIntent after its transaction commits.
	CaptureInTx(ctx context.Context, tx *gorm.DB, req CaptureRequest) (*CaptureResult, error)

	// FinishMoMoIntent calls the MoMo gateway for a persisted PENDING intent
	// and stores the pay_url. Gateway failure returns an error WITHOUT marking
	// the row FAILED — the intent stays retryable (DTM re-drives the branch).
	FinishMoMoIntent(ctx context.Context, paymentID string, req CaptureRequest) (*CaptureResult, error)
}

type captureUsecase struct {
	db          *gorm.DB
	walletRepo  repository.WalletRepository
	ledgerRepo  repository.LedgerRepository
	paymentRepo repository.PaymentRepository
	outboxRepo  repository.OutboxRepository
	momoClient  *momo.Client
	momoBaseURL string // for redirect/ipn construction
	ipnURL      string
	redirectURL string
}

func NewCaptureUsecase(
	db *gorm.DB,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	paymentRepo repository.PaymentRepository,
	outboxRepo repository.OutboxRepository,
	momoClient *momo.Client,
	ipnURL, redirectURL string,
) CaptureUsecase {
	return &captureUsecase{
		db:          db,
		walletRepo:  walletRepo,
		ledgerRepo:  ledgerRepo,
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
		momoClient:  momoClient,
		ipnURL:      ipnURL,
		redirectURL: redirectURL,
	}
}

func (uc *captureUsecase) Capture(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	// Idempotency: return existing payment if already captured for this order.
	existing, err := uc.paymentRepo.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// A MoMo intent row without a pay_url means the gateway call never
		// completed — finish it so retried captures (e.g. a DTM branch being
		// re-driven) still end up with a usable checkout link.
		if existing.Method == entity.MethodMoMo && existing.Status == entity.PaymentPending && existing.PayURL == "" {
			return uc.finishMoMoOrFail(ctx, existing.ID, req)
		}
		return &CaptureResult{
			PaymentID: existing.ID,
			Status:    existing.Status,
			PayURL:    existing.PayURL,
		}, nil
	}

	switch req.Method {
	case entity.MethodWallet:
		return uc.captureWallet(ctx, req)
	case entity.MethodMoMo:
		return uc.captureMoMo(ctx, req)
	default:
		// COD: record intent, no ledger movement.
		return uc.captureCOD(ctx, req)
	}
}

// CaptureInTx is the DTM-branch variant of Capture: all database writes join
// the caller's transaction so the barrier row commits atomically with them.
func (uc *captureUsecase) CaptureInTx(ctx context.Context, tx *gorm.DB, req CaptureRequest) (*CaptureResult, error) {
	existing, err := uc.paymentRepo.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Finishing a dangling MoMo intent is the caller's job (outside tx).
		return &CaptureResult{
			PaymentID: existing.ID,
			Status:    existing.Status,
			PayURL:    existing.PayURL,
		}, nil
	}

	switch req.Method {
	case entity.MethodWallet:
		paymentID, err := uc.captureWalletInTx(ctx, tx, req)
		if err != nil {
			// Bubble up raw — the handler maps ErrInsufficientBalance to a saga
			// abort and anything else to a retryable branch error.
			return nil, err
		}
		return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentCaptured}, nil
	case entity.MethodMoMo:
		paymentID, err := uc.createIntentInTx(ctx, tx, req, entity.MethodMoMo, entity.ProviderMoMo, true)
		if err != nil {
			return nil, err
		}
		return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentPending}, nil
	default:
		paymentID, err := uc.createIntentInTx(ctx, tx, req, entity.MethodCOD, entity.ProviderInternal, false)
		if err != nil {
			return nil, err
		}
		return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentPending}, nil
	}
}

// captureWallet debits the customer wallet synchronously and marks CAPTURED.
func (uc *captureUsecase) captureWallet(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	var paymentID string
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		id, err := uc.captureWalletInTx(ctx, tx, req)
		paymentID = id
		return err
	})

	if txErr != nil {
		slog.ErrorContext(ctx, "capture(wallet): tx failed", "order_id", req.OrderID, "err", txErr)
		return &CaptureResult{Status: entity.PaymentFailed, Error: txErr.Error()}, nil
	}

	slog.InfoContext(ctx, "capture(wallet): captured", "order_id", req.OrderID, "payment_id", paymentID)
	return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentCaptured}, nil
}

// captureWalletInTx performs the wallet debit, payment row and captured event
// inside the given transaction.
func (uc *captureUsecase) captureWalletInTx(ctx context.Context, tx *gorm.DB, req CaptureRequest) (string, error) {
	traceID := outbox.TraceIDFromCtx(ctx)

	// Double-entry: debit CUSTOMER, credit SYSTEM.
	if err := doubleEntryTransfer(ctx, transferParams{
		tx:              tx,
		walletRepo:      uc.walletRepo,
		ledgerRepo:      uc.ledgerRepo,
		debitOwnerType:  entity.WalletOwnerCustomer,
		debitOwnerID:    req.CustomerID,
		creditOwnerType: entity.WalletOwnerSystem,
		creditOwnerID:   entity.SystemOwnerID,
		entryType:       entity.LedgerPayment,
		refType:         entity.LedgerRefOrder,
		refID:           req.OrderID,
		amount:          req.Amount,
		traceID:         traceID,
	}); err != nil {
		return "", err
	}

	// Record payment as CAPTURED.
	p := &entity.Payment{
		OrderID:    &req.OrderID,
		CustomerID: req.CustomerID,
		Method:     entity.MethodWallet,
		Provider:   entity.ProviderInternal,
		Env:        "demo",
		Amount:     req.Amount,
		Status:     entity.PaymentCaptured,
	}
	if err := uc.paymentRepo.Create(ctx, tx, p); err != nil {
		return "", err
	}

	// Publish payment.captured via outbox.
	payload, _ := json.Marshal(map[string]any{
		"payment_id": p.ID,
		"order_id":   req.OrderID,
		"amount":     req.Amount,
		"method":     string(entity.MethodWallet),
	})
	if err := uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
		AggregateType: "payment",
		AggregateID:   p.ID,
		EventType:     "payment.captured",
		Payload:       payload,
		TraceID:       strPtr(traceID),
	}); err != nil {
		return "", err
	}
	return p.ID, nil
}

// createIntentInTx persists a PENDING payment row (MoMo or COD intent) inside
// the given transaction. MoMo rows get the idempotency key the IPN handler
// resolves payments by.
func (uc *captureUsecase) createIntentInTx(
	ctx context.Context, tx *gorm.DB, req CaptureRequest,
	method entity.PaymentMethod, provider entity.PaymentProvider, withIdempotencyKey bool,
) (string, error) {
	p := &entity.Payment{
		OrderID:    &req.OrderID,
		CustomerID: req.CustomerID,
		Method:     method,
		Provider:   provider,
		Env:        "demo",
		Amount:     req.Amount,
		Status:     entity.PaymentPending,
	}
	if withIdempotencyKey {
		p.IdempotencyKey = strPtr(fmt.Sprintf("order:%s", req.OrderID))
	}
	if err := uc.paymentRepo.Create(ctx, tx, p); err != nil {
		return "", err
	}
	return p.ID, nil
}

// captureMoMo creates a MoMo payment intent (PENDING) and returns the pay_url.
// The capture completes asynchronously via the MoMo IPN callback.
func (uc *captureUsecase) captureMoMo(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	var paymentID string

	// Persist PENDING payment first so IPN can find it by idempotency_key.
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		id, err := uc.createIntentInTx(ctx, tx, req, entity.MethodMoMo, entity.ProviderMoMo, true)
		paymentID = id
		return err
	})
	if txErr != nil {
		return nil, txErr
	}

	return uc.finishMoMoOrFail(ctx, paymentID, req)
}

// FinishMoMoIntent calls the MoMo gateway for an already-persisted PENDING row
// and stores the returned pay_url. Runs outside any transaction — a network
// failure must not roll back the intent row. The row is NOT marked FAILED on
// gateway error, so a retried call (DTM re-driving the branch) completes it.
func (uc *captureUsecase) FinishMoMoIntent(ctx context.Context, paymentID string, req CaptureRequest) (*CaptureResult, error) {
	requestID := uuid.NewString()
	result, err := uc.momoClient.CreatePayment(ctx, req.OrderID, "VeloxFood order", req.Amount, uc.redirectURL, uc.ipnURL, requestID)
	if err != nil {
		slog.ErrorContext(ctx, "capture(momo): create-payment failed", "order_id", req.OrderID, "err", err)
		return nil, err
	}

	if err := uc.paymentRepo.UpdatePayURL(ctx, paymentID, result.PayURL); err != nil {
		slog.ErrorContext(ctx, "capture(momo): persist pay_url failed", "payment_id", paymentID, "err", err)
	}

	slog.InfoContext(ctx, "capture(momo): pending, pay_url generated", "order_id", req.OrderID, "payment_id", paymentID)
	return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentPending, PayURL: result.PayURL}, nil
}

// finishMoMoOrFail preserves the synchronous (inline-saga) contract: a gateway
// failure marks the intent FAILED and reports a failed capture immediately.
func (uc *captureUsecase) finishMoMoOrFail(ctx context.Context, paymentID string, req CaptureRequest) (*CaptureResult, error) {
	result, err := uc.FinishMoMoIntent(ctx, paymentID, req)
	if err != nil {
		_ = uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return uc.paymentRepo.UpdateStatus(ctx, tx, paymentID, entity.PaymentFailed, nil)
		})
		return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentFailed, Error: err.Error()}, nil
	}
	return result, nil
}

// captureCOD records a COD payment (saga typically doesn't call Capture for COD,
// but we handle it gracefully with a PENDING record and no ledger movement).
func (uc *captureUsecase) captureCOD(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	var paymentID string
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		id, err := uc.createIntentInTx(ctx, tx, req, entity.MethodCOD, entity.ProviderInternal, false)
		paymentID = id
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentPending}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
