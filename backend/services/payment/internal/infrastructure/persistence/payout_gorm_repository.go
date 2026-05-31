package persistence

import (
	"context"
	"errors"
	"time"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
)

type payoutGormRepository struct {
	db *gorm.DB
}

func NewPayoutGormRepository(db *gorm.DB) repository.PayoutRepository {
	return &payoutGormRepository{db: db}
}

func (r *payoutGormRepository) Create(ctx context.Context, p *entity.PayoutBatch) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *payoutGormRepository) GetByID(ctx context.Context, id string) (*entity.PayoutBatch, error) {
	var p entity.PayoutBatch
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *payoutGormRepository) ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.PayoutBatch, error) {
	var batches []*entity.PayoutBatch
	err := r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&batches).Error
	return batches, err
}

func (r *payoutGormRepository) MarkSettled(ctx context.Context, tx *gorm.DB, batchID, settledByUserID string) error {
	now := time.Now()
	return tx.WithContext(ctx).
		Model(&entity.PayoutBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]any{
			"status":     entity.PayoutSettled,
			"settled_by": settledByUserID,
			"settled_at": now,
		}).Error
}
