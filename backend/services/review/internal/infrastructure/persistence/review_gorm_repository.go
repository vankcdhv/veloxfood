package persistence

import (
	"context"
	"errors"

	"project/services/review/internal/entity"
	"project/services/review/internal/repository"

	"gorm.io/gorm"
)

type reviewGormRepository struct{ db *gorm.DB }

// NewReviewGormRepository returns a ReviewRepository backed by GORM.
func NewReviewGormRepository(db *gorm.DB) repository.ReviewRepository {
	return &reviewGormRepository{db: db}
}

func (r *reviewGormRepository) Create(ctx context.Context, tx *gorm.DB, rev *entity.Review) error {
	return tx.WithContext(ctx).Create(rev).Error
}

func (r *reviewGormRepository) GetByID(ctx context.Context, id string) (*entity.Review, error) {
	var rev entity.Review
	if err := r.db.WithContext(ctx).First(&rev, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrReviewNotFound
		}
		return nil, err
	}
	return &rev, nil
}

func (r *reviewGormRepository) GetByOrderAndTarget(ctx context.Context, orderID, targetType, targetID string) (*entity.Review, error) {
	var rev entity.Review
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND target_type = ? AND target_id = ?", orderID, targetType, targetID).
		First(&rev).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrReviewNotFound
		}
		return nil, err
	}
	return &rev, nil
}

func (r *reviewGormRepository) ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.Review, int64, error) {
	var rows []*entity.Review
	var total int64
	base := r.db.WithContext(ctx).Model(&entity.Review{}).Where("store_id = ? AND status = 'VISIBLE'", storeID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *reviewGormRepository) ListByTarget(ctx context.Context, targetType, targetID string, limit, offset int) ([]*entity.Review, int64, error) {
	var rows []*entity.Review
	var total int64
	base := r.db.WithContext(ctx).Model(&entity.Review{}).
		Where("target_type = ? AND target_id = ? AND status = 'VISIBLE'", targetType, targetID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *reviewGormRepository) RatingSummaryByTarget(ctx context.Context, targetType, targetID string) (float64, int32, error) {
	var result struct {
		Avg   float64
		Count int32
	}
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(AVG(rating), 0) AS avg, COUNT(*) AS count
		   FROM reviews WHERE target_type = ? AND target_id = ? AND status = 'VISIBLE'`,
		targetType, targetID,
	).Scan(&result).Error
	return result.Avg, result.Count, err
}

func (r *reviewGormRepository) ItemRatingSummaries(ctx context.Context, storeID string) ([]repository.ItemRatingSummary, error) {
	// Non-nil so an empty result marshals as [] (not null) for the storefront.
	rows := make([]repository.ItemRatingSummary, 0)
	err := r.db.WithContext(ctx).Raw(
		`SELECT target_id AS target_id, COALESCE(AVG(rating), 0) AS avg, COUNT(*) AS count
		   FROM reviews
		  WHERE store_id = ? AND target_type = 'ITEM' AND status = 'VISIBLE'
		  GROUP BY target_id`, storeID,
	).Scan(&rows).Error
	return rows, err
}

func (r *reviewGormRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Review{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *reviewGormRepository) RatingSummary(ctx context.Context, storeID string) (float64, int32, error) {
	var result struct {
		Avg   float64
		Count int32
	}
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(AVG(rating), 0) AS avg, COUNT(*) AS count
		   FROM reviews WHERE store_id = ? AND status = 'VISIBLE'`, storeID,
	).Scan(&result).Error
	return result.Avg, result.Count, err
}
