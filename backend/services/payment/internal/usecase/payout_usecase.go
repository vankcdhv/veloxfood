package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreatePayoutRequest carries input for creating a payout batch.
type CreatePayoutRequest struct {
	StoreID     string
	PeriodFrom  string // YYYY-MM-DD
	PeriodTo    string // YYYY-MM-DD
	OrderIDs    []string
	TotalAmount int64
}

// PayoutUsecase manages payout batch lifecycle.
type PayoutUsecase interface {
	CreateBatch(ctx context.Context, req CreatePayoutRequest) (*entity.PayoutBatch, error)
	ExecuteBatch(ctx context.Context, batchID, adminUserID string) error
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.PayoutBatch, error)
}

type payoutUsecase struct {
	db          *gorm.DB
	walletRepo  repository.WalletRepository
	ledgerRepo  repository.LedgerRepository
	payoutRepo  repository.PayoutRepository
	outboxRepo  repository.OutboxRepository
}

func NewPayoutUsecase(
	db *gorm.DB,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	payoutRepo repository.PayoutRepository,
	outboxRepo repository.OutboxRepository,
) PayoutUsecase {
	return &payoutUsecase{
		db:         db,
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
		payoutRepo: payoutRepo,
		outboxRepo: outboxRepo,
	}
}

func (uc *payoutUsecase) CreateBatch(ctx context.Context, req CreatePayoutRequest) (*entity.PayoutBatch, error) {
	orderIDsJSON, _ := json.Marshal(req.OrderIDs)
	batch := &entity.PayoutBatch{
		StoreID:     req.StoreID,
		OrderIDs:    orderIDsJSON,
		TotalAmount: req.TotalAmount,
		Status:      entity.PayoutPending,
	}

	if err := uc.payoutRepo.Create(ctx, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (uc *payoutUsecase) ExecuteBatch(ctx context.Context, batchID, adminUserID string) error {
	batch, err := uc.payoutRepo.GetByID(ctx, batchID)
	if err != nil {
		return err
	}
	if batch == nil {
		return ErrPayoutNotFound
	}
	// Idempotent: already SETTLED is a no-op.
	if batch.Status == entity.PayoutSettled {
		return nil
	}

	traceID := outbox.TraceIDFromCtx(ctx)

	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the batch row to prevent concurrent execute calls.
		var locked entity.PayoutBatch
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", batchID).
			First(&locked).Error; err != nil {
			return err
		}
		if locked.Status == entity.PayoutSettled {
			return nil // idempotent — another goroutine settled it
		}

		// Debit STORE_PAYABLE, credit SYSTEM.
		if err := doubleEntryTransfer(ctx, transferParams{
			tx:              tx,
			walletRepo:      uc.walletRepo,
			ledgerRepo:      uc.ledgerRepo,
			debitOwnerType:  entity.WalletOwnerStorePayable,
			debitOwnerID:    batch.StoreID,
			creditOwnerType: entity.WalletOwnerSystem,
			creditOwnerID:   entity.SystemOwnerID,
			entryType:       entity.LedgerPayout,
			refType:         entity.LedgerRefPayoutBatch,
			refID:           batchID,
			amount:          batch.TotalAmount,
			traceID:         traceID,
		}); err != nil {
			return err
		}

		if err := uc.payoutRepo.MarkSettled(ctx, tx, batchID, adminUserID); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"batch_id":     batchID,
			"store_id":     batch.StoreID,
			"total_amount": batch.TotalAmount,
			"settled_by":   adminUserID,
		})
		slog.InfoContext(ctx, "payout: batch settled", "batch_id", batchID, "store_id", batch.StoreID)
		return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "payout",
			AggregateID:   batchID,
			EventType:     "payout.settled",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		})
	})
}

func (uc *payoutUsecase) ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.PayoutBatch, error) {
	return uc.payoutRepo.ListByStore(ctx, storeID, limit, offset)
}
