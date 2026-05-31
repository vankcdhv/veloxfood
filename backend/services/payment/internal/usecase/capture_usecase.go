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
		return &CaptureResult{
			PaymentID: existing.ID,
			Status:    existing.Status,
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

// captureWallet debits the customer wallet synchronously and marks CAPTURED.
func (uc *captureUsecase) captureWallet(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	traceID := outbox.TraceIDFromCtx(ctx)
	var paymentID string

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
			return err
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
			return err
		}
		paymentID = p.ID

		// Publish payment.captured via outbox.
		payload, _ := json.Marshal(map[string]any{
			"payment_id": p.ID,
			"order_id":   req.OrderID,
			"amount":     req.Amount,
			"method":     string(entity.MethodWallet),
		})
		return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "payment",
			AggregateID:   p.ID,
			EventType:     "payment.captured",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		})
	})

	if txErr != nil {
		slog.ErrorContext(ctx, "capture(wallet): tx failed", "order_id", req.OrderID, "err", txErr)
		return &CaptureResult{Status: entity.PaymentFailed, Error: txErr.Error()}, nil
	}

	slog.InfoContext(ctx, "capture(wallet): captured", "order_id", req.OrderID, "payment_id", paymentID)
	return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentCaptured}, nil
}

// captureMoMo creates a MoMo payment intent (PENDING) and returns the pay_url.
// The capture completes asynchronously via the MoMo IPN callback.
func (uc *captureUsecase) captureMoMo(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	requestID := uuid.NewString()
	var paymentID string
	var payURL string

	// Persist PENDING payment first so IPN can find it by idempotency_key.
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		iKey := fmt.Sprintf("order:%s", req.OrderID)
		p := &entity.Payment{
			OrderID:        &req.OrderID,
			CustomerID:     req.CustomerID,
			Method:         entity.MethodMoMo,
			Provider:       entity.ProviderMoMo,
			Env:            "demo",
			Amount:         req.Amount,
			Status:         entity.PaymentPending,
			IdempotencyKey: strPtr(iKey),
		}
		if err := uc.paymentRepo.Create(ctx, tx, p); err != nil {
			return err
		}
		paymentID = p.ID
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	// Call MoMo outside the transaction — network failure should not roll back the DB row.
	result, err := uc.momoClient.CreatePayment(ctx, req.OrderID, "VeloxFood order", req.Amount, uc.redirectURL, uc.ipnURL, requestID)
	if err != nil {
		slog.ErrorContext(ctx, "capture(momo): create-payment failed", "order_id", req.OrderID, "err", err)
		// Mark FAILED and propagate.
		_ = uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return uc.paymentRepo.UpdateStatus(ctx, tx, paymentID, entity.PaymentFailed, nil)
		})
		return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentFailed, Error: err.Error()}, nil
	}
	payURL = result.PayURL

	slog.InfoContext(ctx, "capture(momo): pending, pay_url generated", "order_id", req.OrderID, "payment_id", paymentID)
	return &CaptureResult{PaymentID: paymentID, Status: entity.PaymentPending, PayURL: payURL}, nil
}

// captureCOD records a COD payment (saga typically doesn't call Capture for COD,
// but we handle it gracefully with a PENDING record and no ledger movement).
func (uc *captureUsecase) captureCOD(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	var paymentID string
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p := &entity.Payment{
			OrderID:    &req.OrderID,
			CustomerID: req.CustomerID,
			Method:     entity.MethodCOD,
			Provider:   entity.ProviderInternal,
			Env:        "demo",
			Amount:     req.Amount,
			Status:     entity.PaymentPending,
		}
		if err := uc.paymentRepo.Create(ctx, tx, p); err != nil {
			return err
		}
		paymentID = p.ID
		return nil
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
