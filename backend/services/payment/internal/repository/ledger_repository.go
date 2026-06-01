package repository

import (
	"context"
	"time"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// SettleableOrder is a projection of a STORE_PAYABLE credit ledger entry
// grouped by the order it references.
type SettleableOrder struct {
	OrderID   string
	Amount    int64
	CreatedAt time.Time
}

// LedgerRepository manages append-only ledger entries.
type LedgerRepository interface {
	// Append inserts a new ledger entry within the provided transaction.
	Append(ctx context.Context, tx *gorm.DB, entry *entity.LedgerEntry) error

	// ListByWallet returns ledger entries for a wallet, newest-first.
	ListByWallet(ctx context.Context, walletID string, limit, offset int) ([]*entity.LedgerEntry, error)

	// SettleableOrdersForStore returns STORE_PAYABLE credit entries (ref_type=order,
	// amount>0) for the given store that are NOT already included in a SETTLED
	// payout batch for that store.
	SettleableOrdersForStore(ctx context.Context, storeID string) ([]*SettleableOrder, error)
}
