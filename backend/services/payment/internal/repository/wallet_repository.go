package repository

import (
	"context"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// WalletRepository manages wallet records.
// GetOrCreate uses FOR UPDATE to prevent races on concurrent first-access.
type WalletRepository interface {
	// GetOrCreateForUpdate fetches the wallet matching (ownerType, ownerID),
	// creating it if absent. The row is locked FOR UPDATE within tx.
	GetOrCreateForUpdate(ctx context.Context, tx *gorm.DB, ownerType entity.WalletOwnerType, ownerID string) (*entity.Wallet, error)

	// UpdateBalance persists a new balance on the wallet within tx.
	UpdateBalance(ctx context.Context, tx *gorm.DB, walletID string, newBalance int64) error

	// GetByOwner returns the wallet for (ownerType, ownerID) without locking.
	GetByOwner(ctx context.Context, ownerType entity.WalletOwnerType, ownerID string) (*entity.Wallet, error)
}
