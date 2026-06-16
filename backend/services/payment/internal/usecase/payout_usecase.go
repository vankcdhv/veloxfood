package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

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

// parseDateOr parses a YYYY-MM-DD string, returning fallback on empty/invalid input.
func parseDateOr(s string, fallback time.Time) time.Time {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return fallback
}

func (uc *payoutUsecase) CreateBatch(ctx context.Context, req CreatePayoutRequest) (*entity.PayoutBatch, error) {
	// Always re-derive amounts server-side from the settleable summary — never
	// trust client totals. The client MAY pass order_ids to pay out only a subset
	// (period/per-order filter); we keep only ids that are genuinely settleable
	// and sum their server-side amounts, so a client can neither inflate the
	// total nor include already-settled / foreign orders.
	summary, err := uc.GetSettleableSummary(ctx, req.StoreID)
	if err != nil {
		return nil, err
	}
	settleableByID := make(map[string]int64, len(summary.Orders))
	for _, o := range summary.Orders {
		settleableByID[o.OrderID] = o.Amount
	}

	var orderIDs []string
	var total int64
	if len(req.OrderIDs) > 0 {
		// Subset selected by the admin — intersect with the settleable set.
		for _, id := range req.OrderIDs {
			if amt, ok := settleableByID[id]; ok {
				orderIDs = append(orderIDs, id)
				total += amt
			}
		}
	} else {
		// No selection → pay out everything settleable (back-compat).
		for _, o := range summary.Orders {
			orderIDs = append(orderIDs, o.OrderID)
			total += o.Amount
		}
	}
	if total <= 0 || len(orderIDs) == 0 {
		return nil, ErrNothingToSettle
	}
	orderIDsJSON, _ := json.Marshal(orderIDs)
	// Persist the batch period; fall back to today when the client omits it so
	// the history never shows a zero date (0001-01-01).
	periodFrom := parseDateOr(req.PeriodFrom, time.Now().UTC())
	periodTo := parseDateOr(req.PeriodTo, periodFrom)
	batch := &entity.PayoutBatch{
		StoreID:     req.StoreID,
		PeriodFrom:  periodFrom,
		PeriodTo:    periodTo,
		OrderIDs:    orderIDsJSON,
		TotalAmount: total,
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
