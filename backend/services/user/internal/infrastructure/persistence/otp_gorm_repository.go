package persistence

import (
	"context"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type otpGormRepository struct {
	db *gorm.DB
}

func NewOTPGormRepository(db *gorm.DB) repository.OTPRepository {
	return &otpGormRepository{db: db}
}

func (r *otpGormRepository) Create(ctx context.Context, otp *entity.OTPCode) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

// FindActive finds an unexpired, unused OTP for the destination+purpose combo.
func (r *otpGormRepository) FindActive(ctx context.Context, destination string, purpose entity.OTPPurpose) (*entity.OTPCode, error) {
	var otp entity.OTPCode
	err := r.db.WithContext(ctx).
		Where("destination = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?",
			destination, purpose, time.Now()).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *otpGormRepository) IncrementAttempts(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.OTPCode{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *otpGormRepository) MarkUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.OTPCode{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}

func (r *otpGormRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entity.OTPCode{}).Error
}
