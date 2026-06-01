package persistence

import (
	"context"
	"errors"

	"project/services/review/internal/entity"
	"project/services/review/internal/repository"

	"gorm.io/gorm"
)

// ── ReviewReply ──────────────────────────────────────────────────────────────

type reviewReplyGormRepository struct{ db *gorm.DB }

func NewReviewReplyGormRepository(db *gorm.DB) repository.ReviewReplyRepository {
	return &reviewReplyGormRepository{db: db}
}

func (r *reviewReplyGormRepository) Create(ctx context.Context, reply *entity.ReviewReply) error {
	return r.db.WithContext(ctx).Create(reply).Error
}

func (r *reviewReplyGormRepository) GetByReviewID(ctx context.Context, reviewID string) (*entity.ReviewReply, error) {
	var reply entity.ReviewReply
	err := r.db.WithContext(ctx).Where("review_id = ?", reviewID).First(&reply).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // no reply yet
		}
		return nil, err
	}
	return &reply, nil
}

// ── ReviewReport ─────────────────────────────────────────────────────────────

type reviewReportGormRepository struct{ db *gorm.DB }

func NewReviewReportGormRepository(db *gorm.DB) repository.ReviewReportRepository {
	return &reviewReportGormRepository{db: db}
}

func (r *reviewReportGormRepository) Create(ctx context.Context, report *entity.ReviewReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *reviewReportGormRepository) ListOpen(ctx context.Context, limit, offset int) ([]*entity.ReviewReport, int64, error) {
	var rows []*entity.ReviewReport
	var total int64
	base := r.db.WithContext(ctx).Model(&entity.ReviewReport{}).Where("status = 'OPEN'")
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *reviewReportGormRepository) Resolve(ctx context.Context, reviewID string) error {
	return r.db.WithContext(ctx).
		Model(&entity.ReviewReport{}).
		Where("review_id = ? AND status = 'OPEN'", reviewID).
		Update("status", "RESOLVED").Error
}
