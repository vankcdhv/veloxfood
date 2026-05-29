package persistence

import (
	"context"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type passwordResetGormRepository struct {
	db *gorm.DB
}

func NewPasswordResetGormRepository(db *gorm.DB) repository.PasswordResetRepository {
	return &passwordResetGormRepository{db: db}
}

func (r *passwordResetGormRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *passwordResetGormRepository) FindByTokenHash(ctx context.Context, hash string) (*entity.PasswordResetToken, error) {
	var token entity.PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now()).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *passwordResetGormRepository) MarkUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}
