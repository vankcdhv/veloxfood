package repository

import (
	"context"

	"project/services/notification/internal/entity"

	"gorm.io/gorm"
)

// NotificationRepository is the persistence contract for notifications.
type NotificationRepository interface {
	// Create inserts a new notification row.
	Create(ctx context.Context, n *entity.Notification) error

	// ListByUser returns paginated notifications for a user, newest first.
	ListByUser(ctx context.Context, userID string, skip, limit int) ([]*entity.Notification, int64, error)

	// UnreadCount returns the number of unread notifications for a user.
	UnreadCount(ctx context.Context, userID string) (int64, error)

	// MarkRead sets read_at = NOW() for a single notification owned by userID.
	// Returns false if the row does not exist or belongs to another user.
	MarkRead(ctx context.Context, id, userID string) (bool, error)

	// MarkAllRead sets read_at = NOW() on all unread notifications for userID.
	MarkAllRead(ctx context.Context, userID string) error
}

// DeviceTokenRepository manages FCM device registrations.
type DeviceTokenRepository interface {
	// Upsert inserts or ignores a device token for a user.
	Upsert(ctx context.Context, token *entity.DeviceToken) error

	// Delete removes a device token by ID for a given user.
	// Returns false if not found or owned by another user.
	Delete(ctx context.Context, id, userID string) (bool, error)

	// ListByUser returns all FCM tokens registered for a user.
	ListByUser(ctx context.Context, userID string) ([]*entity.DeviceToken, error)
}

// ProcessedEventRepository deduplicates incoming Kafka events.
type ProcessedEventRepository interface {
	// MarkProcessed inserts event_id. Returns (true, nil) on first insert,
	// (false, nil) on duplicate, or (false, err) on unexpected failure.
	MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (bool, error)
}
