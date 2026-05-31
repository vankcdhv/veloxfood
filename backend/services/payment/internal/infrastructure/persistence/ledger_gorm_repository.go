package persistence

import (
	"context"

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
