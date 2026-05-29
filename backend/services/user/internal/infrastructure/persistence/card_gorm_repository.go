package persistence

import (
	"context"
	"errors"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type cardGormRepository struct {
	db *gorm.DB
}

// NewCardGormRepository constructs a GORM-backed CardRepository.
func NewCardGormRepository(db *gorm.DB) repository.CardRepository {
	return &cardGormRepository{db: db}
}

func (r *cardGormRepository) Create(ctx context.Context, card *entity.CardIdentifier) error {
	return r.db.WithContext(ctx).Create(card).Error
}

func (r *cardGormRepository) GetByID(ctx context.Context, id string) (*entity.CardIdentifier, error) {
	var card entity.CardIdentifier
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&card).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &card, nil
}

func (r *cardGormRepository) ListByUser(ctx context.Context, userID string, includeRevoked bool) ([]*entity.CardIdentifier, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if !includeRevoked {
		q = q.Where("revoked_at IS NULL")
	}
	var cards []*entity.CardIdentifier
	if err := q.Order("issued_at DESC").Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *cardGormRepository) Revoke(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.CardIdentifier{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *cardGormRepository) CheckActiveDuplicate(ctx context.Context, kind entity.CardKind, identifier string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.CardIdentifier{}).
		Where("kind = ? AND identifier = ? AND revoked_at IS NULL", kind, identifier).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
