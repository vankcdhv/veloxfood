package persistence

import (
	"context"
	"time"

	"project/services/notification/internal/entity"
	"project/services/notification/internal/repository"

	"gorm.io/gorm"
)

type notificationGormRepository struct {
	db *gorm.DB
}

// NewNotificationGormRepository returns a NotificationRepository backed by GORM.
func NewNotificationGormRepository(db *gorm.DB) repository.NotificationRepository {
	return &notificationGormRepository{db: db}
}

func (r *notificationGormRepository) Create(ctx context.Context, n *entity.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationGormRepository) ListByUser(ctx context.Context, userID string, skip, limit int) ([]*entity.Notification, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*entity.Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(skip).Limit(limit).
		Find(&rows).Error
	return rows, total, err
}

func (r *notificationGormRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *notificationGormRepository) MarkRead(ctx context.Context, id, userID string) (bool, error) {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *notificationGormRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}

// compile-time interface check
var _ repository.NotificationRepository = (*notificationGormRepository)(nil)
