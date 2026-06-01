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

// SettleableSummary is the response model for the admin settlements endpoint.
type SettleableSummary struct {
	StoreID        string
	PayableBalance int64
	Orders         []SettleableOrderItem
	Total          int64
}

// SettleableOrderItem is one order contributing to the unsettled balance.
type SettleableOrderItem struct {
	OrderID   string
	Amount    int64
	CreatedAt string // RFC3339
}

// PayoutUsecase manages payout batch lifecycle.
type PayoutUsecase interface {
	CreateBatch(ctx context.Context, req CreatePayoutRequest) (*entity.PayoutBatch, error)
	ExecuteBatch(ctx context.Context, batchID, adminUserID string) error
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.PayoutBatch, error)

	// GetSettleableSummary returns the payable wallet balance and the
	// individual orders behind it for the given store.
	GetSettleableSummary(ctx context.Context, storeID string) (*SettleableSummary, error)
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

func (uc *payoutUsecase) GetSettleableSummary(ctx context.Context, storeID string) (*SettleableSummary, error) {
	// Wallet balance — zero if wallet does not exist yet.
	wallet, err := uc.walletRepo.GetByOwner(ctx, entity.WalletOwnerStorePayable, storeID)
	if err != nil {
		return nil, err
	}
	var balance int64
	if wallet != nil {
		balance = wallet.Balance
	}

	// Orders behind the unsettled balance.
	orders, err := uc.ledgerRepo.SettleableOrdersForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}

	items := make([]SettleableOrderItem, 0, len(orders))
	var total int64
	for _, o := range orders {
		items = append(items, SettleableOrderItem{
			OrderID:   o.OrderID,
			Amount:    o.Amount,
			CreatedAt: o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
		total += o.Amount
	}

	return &SettleableSummary{
		StoreID:        storeID,
		PayableBalance: balance,
		Orders:         items,
		Total:          total,
	}, nil
}
