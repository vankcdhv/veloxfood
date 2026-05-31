package persistence

import (
	"context"
	"errors"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type walletGormRepository struct {
	db *gorm.DB
}

func NewWalletGormRepository(db *gorm.DB) repository.WalletRepository {
	return &walletGormRepository{db: db}
}

// GetOrCreateForUpdate fetches or creates the wallet for (ownerType, ownerID),
// locking the row FOR UPDATE within tx so callers can safely mutate balance.
func (r *walletGormRepository) GetOrCreateForUpdate(
	ctx context.Context, tx *gorm.DB,
	ownerType entity.WalletOwnerType, ownerID string,
) (*entity.Wallet, error) {
	var w entity.Wallet

	// Try to find existing row with row-level lock.
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		First(&w).Error

	if err == nil {
		return &w, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Row absent — insert then re-lock. Use INSERT ... ON CONFLICT DO NOTHING to
	// handle the race where two goroutines both see ErrRecordNotFound simultaneously.
	w = entity.Wallet{OwnerType: ownerType, OwnerID: ownerID, Balance: 0}
	createErr := tx.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&w).Error
	if createErr != nil {
		return nil, createErr
	}

	// Re-read with lock (handles the case where another tx won the INSERT race).
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		First(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *walletGormRepository) UpdateBalance(ctx context.Context, tx *gorm.DB, walletID string, newBalance int64) error {
	return tx.WithContext(ctx).
		Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", newBalance).Error
}

func (r *walletGormRepository) GetByOwner(ctx context.Context, ownerType entity.WalletOwnerType, ownerID string) (*entity.Wallet, error) {
	var w entity.Wallet
	err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}
