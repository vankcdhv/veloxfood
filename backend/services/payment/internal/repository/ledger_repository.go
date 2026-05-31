package repository

import (
	"context"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// LedgerRepository manages append-only ledger entries.
type LedgerRepository interface {
	// Append inserts a new ledger entry within the provided transaction.
	Append(ctx context.Context, tx *gorm.DB, entry *entity.LedgerEntry) error

	// ListByWallet returns ledger entries for a wallet, newest-first.
	ListByWallet(ctx context.Context, walletID string, limit, offset int) ([]*entity.LedgerEntry, error)
}
