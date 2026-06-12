package persistence

import (
	"context"
	"time"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
)

type ledgerGormRepository struct {
	db *gorm.DB
}

func NewLedgerGormRepository(db *gorm.DB) repository.LedgerRepository {
	return &ledgerGormRepository{db: db}
}

func (r *ledgerGormRepository) Append(ctx context.Context, tx *gorm.DB, entry *entity.LedgerEntry) error {
	return tx.WithContext(ctx).Create(entry).Error
}

func (r *ledgerGormRepository) ListByWallet(ctx context.Context, walletID string, limit, offset int) ([]*entity.LedgerEntry, error) {
	var entries []*entity.LedgerEntry
	err := r.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	return entries, err
}

// SettleableOrdersForStore returns distinct order-level STORE_PAYABLE credit
// ledger entries for storeID that have not yet been included in a SETTLED
// payout batch for that store.
//
// Strategy: collect all order_ids already in SETTLED batches for the store,
// then exclude them from the STORE_PAYABLE credit entries query.
// Using a NOT IN subquery keeps it in one round-trip without needing JSONB ops.
func (r *ledgerGormRepository) SettleableOrdersForStore(ctx context.Context, storeID string) ([]*repository.SettleableOrder, error) {
	// Subquery: unnest order_ids from payout_batches already covering this store's
	// orders. Exclude PENDING batches too — once an order is in an unexecuted
	// batch it must not appear as settleable again, otherwise a second batch could
	// be created for the same order and the store would be paid twice.
	// JSONB array → set of text UUIDs, then cast to uuid for the NOT IN check.
	const settledOrdersSubquery = `
		SELECT jsonb_array_elements_text(order_ids)::uuid
		FROM payout_batches
		WHERE store_id = ? AND status IN ('PENDING', 'SETTLED')
	`

	type row struct {
		RefID     string    `gorm:"column:ref_id"`
		Amount    int64     `gorm:"column:amount"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT le.ref_id, le.amount, le.created_at
		FROM ledger_entries le
		JOIN wallets w ON w.id = le.wallet_id
		WHERE w.owner_type = 'STORE_PAYABLE'
		  AND w.owner_id   = ?
		  AND le.entry_type = 'PAYMENT'
		  AND le.ref_type   = 'order'
		  AND le.amount     > 0
		  AND le.ref_id NOT IN (`+settledOrdersSubquery+`)
		ORDER BY le.created_at ASC
	`, storeID, storeID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*repository.SettleableOrder, 0, len(rows))
	for _, row := range rows {
		result = append(result, &repository.SettleableOrder{
			OrderID:   row.RefID,
			Amount:    row.Amount,
			CreatedAt: row.CreatedAt,
		})
	}
	return result, nil
}
