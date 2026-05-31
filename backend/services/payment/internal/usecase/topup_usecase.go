package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/infrastructure/momo"
	"project/services/payment/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TopupResult is the outcome of a wallet top-up initiation.
type TopupResult struct {
	TopupID string
	PayURL  string
}

// TopupUsecase initiates a MoMo wallet top-up. Completion is via IPN.
type TopupUsecase interface {
	InitiateTopup(ctx context.Context, customerID string, amount int64) (*TopupResult, error)
}

type topupUsecase struct {
	db          *gorm.DB
	paymentRepo repository.PaymentRepository
	momoClient  *momo.Client
	ipnURL      string
	redirectURL string
}

func NewTopupUsecase(
	db *gorm.DB,
	paymentRepo repository.PaymentRepository,
	momoClient *momo.Client,
	ipnURL, redirectURL string,
) TopupUsecase {
	return &topupUsecase{
		db:          db,
		paymentRepo: paymentRepo,
		momoClient:  momoClient,
		ipnURL:      ipnURL,
		redirectURL: redirectURL,
	}
}

func (uc *topupUsecase) InitiateTopup(ctx context.Context, customerID string, amount int64) (*TopupResult, error) {
	topupID := uuid.NewString()
	requestID := uuid.NewString()
	iKey := fmt.Sprintf("topup:%s", topupID)

	// Persist PENDING payment before calling MoMo so IPN can match by idempotency_key.
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p := &entity.Payment{
			TopupID:        &topupID,
			CustomerID:     customerID,
			Method:         entity.MethodMoMo,
			Provider:       entity.ProviderMoMo,
			Env:            "demo",
			Amount:         amount,
			Status:         entity.PaymentPending,
			IdempotencyKey: strPtr(iKey),
		}
		return uc.paymentRepo.Create(ctx, tx, p)
	})
	if txErr != nil {
		return nil, txErr
	}

	// Call MoMo outside transaction to avoid holding a DB connection.
	result, err := uc.momoClient.CreatePayment(ctx, topupID, "VeloxFood wallet topup", amount, uc.redirectURL, uc.ipnURL, requestID)
	if err != nil {
		slog.ErrorContext(ctx, "topup: momo create-payment failed", "customer_id", customerID, "err", err)
		return nil, err
	}

	slog.InfoContext(ctx, "topup: MoMo intent created", "topup_id", topupID, "customer_id", customerID)
	return &TopupResult{TopupID: topupID, PayURL: result.PayURL}, nil
}
