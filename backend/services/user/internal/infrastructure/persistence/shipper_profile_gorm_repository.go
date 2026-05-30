package persistence

import (
	"context"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type shipperProfileGormRepository struct {
	db *gorm.DB
}

func NewShipperProfileGormRepository(db *gorm.DB) repository.ShipperProfileRepository {
	return &shipperProfileGormRepository{db: db}
}

func (r *shipperProfileGormRepository) Create(ctx context.Context, p *entity.ShipperProfile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *shipperProfileGormRepository) GetByUserID(ctx context.Context, userID string) (*entity.ShipperProfile, error) {
	var p entity.ShipperProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *shipperProfileGormRepository) UpdateStatus(ctx context.Context, userID string, status entity.ShipperStatus, approvedBy *string, approvedAt *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.ShipperProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"status":      status,
			"approved_by": approvedBy,
			"approved_at": approvedAt,
			"updated_at":  time.Now(),
		}).Error
}

func (r *shipperProfileGormRepository) List(ctx context.Context, status *entity.ShipperStatus, offset, limit int) ([]*entity.ShipperProfile, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.ShipperProfile{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*entity.ShipperProfile
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
