package persistence

import (
	"context"

	"project/services/notification/internal/entity"
	"project/services/notification/internal/repository"

	"gorm.io/gorm"
)

type deviceTokenGormRepository struct {
	db *gorm.DB
}

// NewDeviceTokenGormRepository returns a DeviceTokenRepository backed by GORM.
func NewDeviceTokenGormRepository(db *gorm.DB) repository.DeviceTokenRepository {
	return &deviceTokenGormRepository{db: db}
}

func (r *deviceTokenGormRepository) Upsert(ctx context.Context, token *entity.DeviceToken) error {
	// ON CONFLICT on fcm_token unique index — update user_id and platform in case
	// the token was previously registered by a different user (device swap).
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO device_tokens (user_id, fcm_token, platform)
		      VALUES (?, ?, ?)
		      ON CONFLICT (fcm_token) DO UPDATE
		        SET user_id = EXCLUDED.user_id,
		            platform = EXCLUDED.platform`,
			token.UserID, token.FCMToken, token.Platform,
		).Error
}

func (r *deviceTokenGormRepository) Delete(ctx context.Context, id, userID string) (bool, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&entity.DeviceToken{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *deviceTokenGormRepository) ListByUser(ctx context.Context, userID string) ([]*entity.DeviceToken, error) {
	var rows []*entity.DeviceToken
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

// compile-time interface check
var _ repository.DeviceTokenRepository = (*deviceTokenGormRepository)(nil)
