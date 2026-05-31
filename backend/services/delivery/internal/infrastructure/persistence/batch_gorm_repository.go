package persistence

import (
	"context"
	"errors"
	"fmt"

	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
)

type batchGormRepository struct {
	db *gorm.DB
}

// NewBatchGormRepository returns a BatchRepository backed by GORM.
func NewBatchGormRepository(db *gorm.DB) repository.BatchRepository {
	return &batchGormRepository{db: db}
}

func (r *batchGormRepository) FindOpenBatch(ctx context.Context, tx *gorm.DB, shipperID, storeID string) (*entity.DeliveryBatch, error) {
	var b entity.DeliveryBatch
	err := tx.WithContext(ctx).
		Where("shipper_id = ? AND store_id = ? AND status = ?", shipperID, storeID, entity.BatchOpen).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // no open batch — caller must create one
		}
		return nil, fmt.Errorf("find open batch: %w", err)
	}
	return &b, nil
}

func (r *batchGormRepository) CreateBatch(ctx context.Context, tx *gorm.DB, b *entity.DeliveryBatch) error {
	return tx.WithContext(ctx).Create(b).Error
}
