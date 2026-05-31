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

	"gorm.io/gorm"
)

// IPNRequest carries the parsed fields from a MoMo IPN callback.
type IPNRequest struct {
	PartnerCode  string
	AccessKey    string
	RequestID    string
	Amount       int64
	OrderID      string // maps to topup_id or order_id depending on context
	OrderInfo    string
	OrderType    string
	TransID      int64
	ResultCode   int
	Message      string
	PayType      string
	ResponseTime int64
	ExtraData    string
	Signature    string
}

// IPNUsecase handles MoMo IPN callbacks.
type IPNUsecase interface {
	HandleIPN(ctx context.Context, req IPNRequest) error
}

type ipnUsecase struct {
	db          *gorm.DB
	walletRepo  repository.WalletRepository
	ledgerRepo  repository.LedgerRepository
	paymentRepo repository.PaymentRepository
	outboxRepo  repository.OutboxRepository
	momoClient  *momo.Client
}

func NewIPNUsecase(
	db *gorm.DB,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	paymentRepo repository.PaymentRepository,
	outboxRepo repository.OutboxRepository,
	momoClient *momo.Client,
) IPNUsecase {
	return &ipnUsecase{
		db:          db,
		walletRepo:  walletRepo,
		ledgerRepo:  ledgerRepo,
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
		momoClient:  momoClient,
	}
}

func (uc *ipnUsecase) HandleIPN(ctx context.Context, req IPNRequest) error {
	// Verify HMAC signature first — reject tampered requests before any DB work.
	if err := uc.momoClient.VerifyIPN(momo.IPNFields{
		AccessKey:    req.AccessKey,
		Amount:       req.Amount,
		ExtraData:    req.ExtraData,
		Message:      req.Message,
		OrderID:      req.OrderID,
		OrderInfo:    req.OrderInfo,
		OrderType:    req.OrderType,
		PartnerCode:  req.PartnerCode,
		PayType:      req.PayType,
		RequestID:    req.RequestID,
		ResponseTime: req.ResponseTime,
		ResultCode:   req.ResultCode,
		TransID:      req.TransID,
		Signature:    req.Signature,
	}); err != nil {
		slog.WarnContext(ctx, "ipn: signature verification failed", "order_id", req.OrderID)
		return fmt.Errorf("invalid signature")
	}

	// Only process successful payments (resultCode == 0).
	if req.ResultCode != 0 {
		slog.WarnContext(ctx, "ipn: non-zero result code, ignoring", "order_id", req.OrderID, "code", req.ResultCode)
		return nil
	}

	// Look up payment by idempotency key = "order:<orderID>" or "topup:<topupID>".
	// MoMo sends back the orderId we originally sent, which is either an order UUID or topup UUID.
	iKey := fmt.Sprintf("order:%s", req.OrderID)
	payment, err := uc.paymentRepo.GetByIdempotencyKey(ctx, iKey)
	if err != nil {
		return err
	}

	// Try topup key if not found via order key.
	if payment == nil {
		iKey = fmt.Sprintf("topup:%s", req.OrderID)
		payment, err = uc.paymentRepo.GetByIdempotencyKey(ctx, iKey)
		if err != nil {
			return err
		}
	}

	if payment == nil {
		slog.WarnContext(ctx, "ipn: no payment found for idempotency key", "order_id", req.OrderID)
		return fmt.Errorf("payment not found")
	}

	// Idempotency: already CAPTURED is a no-op.
	if payment.Status == entity.PaymentCaptured {
		slog.InfoContext(ctx, "ipn: already captured, skipping", "payment_id", payment.ID)
		return nil
	}

	transID := fmt.Sprintf("%d", req.TransID)
	traceID := outbox.TraceIDFromCtx(ctx)

	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if payment.TopupID != nil {
			// Top-up: credit CUSTOMER wallet.
			if err := doubleEntryTransfer(ctx, transferParams{
				tx:              tx,
				walletRepo:      uc.walletRepo,
				ledgerRepo:      uc.ledgerRepo,
				debitOwnerType:  entity.WalletOwnerSystem,
				debitOwnerID:    entity.SystemOwnerID,
				creditOwnerType: entity.WalletOwnerCustomer,
				creditOwnerID:   payment.CustomerID,
				entryType:       entity.LedgerTopup,
				refType:         entity.LedgerRefTopup,
				refID:           *payment.TopupID,
				amount:          payment.Amount,
				traceID:         traceID,
			}); err != nil {
				return err
			}

			if err := uc.paymentRepo.UpdateStatus(ctx, tx, payment.ID, entity.PaymentCaptured, &transID); err != nil {
				return err
			}

			payload, _ := json.Marshal(map[string]any{
				"payment_id":  payment.ID,
				"topup_id":    *payment.TopupID,
				"customer_id": payment.CustomerID,
				"amount":      payment.Amount,
			})
			return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
				AggregateType: "wallet",
				AggregateID:   payment.CustomerID,
				EventType:     "wallet.topped_up",
				Payload:       payload,
				TraceID:       strPtr(traceID),
			})
		}

		// Order payment: credit SYSTEM wallet (customer already charged async).
		if payment.OrderID != nil {
			if err := doubleEntryTransfer(ctx, transferParams{
				tx:              tx,
				walletRepo:      uc.walletRepo,
				ledgerRepo:      uc.ledgerRepo,
				debitOwnerType:  entity.WalletOwnerSystem,
				debitOwnerID:    entity.SystemOwnerID,
				creditOwnerType: entity.WalletOwnerSystem,
				creditOwnerID:   entity.SystemOwnerID,
				entryType:       entity.LedgerPayment,
				refType:         entity.LedgerRefOrder,
				refID:           *payment.OrderID,
				amount:          payment.Amount,
				traceID:         traceID,
			}); err != nil {
				// SYSTEM→SYSTEM is a no-op in terms of balance but we still record entries.
				// If it errors for some reason, log and continue with status update.
				slog.WarnContext(ctx, "ipn: order ledger entry failed (non-fatal)", "err", err)
			}

			if err := uc.paymentRepo.UpdateStatus(ctx, tx, payment.ID, entity.PaymentCaptured, &transID); err != nil {
				return err
			}

			payload, _ := json.Marshal(map[string]any{
				"payment_id": payment.ID,
				"order_id":   *payment.OrderID,
				"amount":     payment.Amount,
				"method":     string(entity.MethodMoMo),
			})
			return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
				AggregateType: "payment",
				AggregateID:   payment.ID,
				EventType:     "payment.captured",
				Payload:       payload,
				TraceID:       strPtr(traceID),
			})
		}

		return nil
	})
}
