package persistence

import (
	"context"
	"time"

	"project/services/order/internal/entity"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
)

type compensationGormRepository struct {
	db *gorm.DB
}

func NewCompensationGormRepository(db *gorm.DB) repository.CompensationRepository {
	return &compensationGormRepository{db: db}
}

func (r *compensationGormRepository) Create(ctx context.Context, c *entity.PendingCompensation) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *compensationGormRepository) ListOpen(ctx context.Context, maxAttempts, limit int) ([]*entity.PendingCompensation, error) {
	var rows []*entity.PendingCompensation
	err := r.db.WithContext(ctx).
		Where("done_at IS NULL AND attempts < ?", maxAttempts).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *compensationGormRepository) MarkDone(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.PendingCompensation{}).
		Where("id = ?", id).
		Update("done_at", now).Error
}

func (r *compensationGormRepository) MarkFailed(ctx context.Context, id, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&entity.PendingCompensation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": errMsg,
		}).Error
}
