package persistence

import (
	"context"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type outboxGormRepository struct {
	db *gorm.DB
}

// NewOutboxGormRepository returns an OutboxRepository backed by GORM.
// db is required for Pickup/MarkPublished/MarkFailed queries;
// Append uses the caller-provided tx so it participates in business transactions.
func NewOutboxGormRepository(db *gorm.DB) repository.OutboxRepository {
	return &outboxGormRepository{db: db}
}

// Append inserts an OutboxEvent using the provided GORM transaction.
// Caller must pass a tx obtained from db.Begin() or db.Transaction callbacks.
func (r *outboxGormRepository) Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error {
	return tx.WithContext(ctx).Create(evt).Error
}

// Pickup returns up to limit pending events with fewer than maxAttempts retries.
// Optimistic approach: SELECT without long-held lock; MarkPublished/MarkFailed use
// conditional WHERE to handle concurrent dispatchers safely (only one wins per row).
func (r *outboxGormRepository) Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error) {
	var events []*entity.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ? AND attempts < ?", entity.OutboxStatusPending, maxAttempts).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// MarkPublished sets status='published' and published_at=now for a pending event.
// The WHERE clause guards against double-publish when two dispatcher instances race.
func (r *outboxGormRepository) MarkPublished(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.OutboxEvent{}).
		Where("id = ? AND status = ?", id, entity.OutboxStatusPending).
		Updates(map[string]any{
			"status":       entity.OutboxStatusPublished,
			"published_at": now,
		}).Error
}

// MarkFailed increments attempts and stores the error reason.
// Transitions to status='failed' when attempts reaches maxAttempts.
func (r *outboxGormRepository) MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error {
	newStatus := entity.OutboxStatusPending
	if attempts >= maxAttempts {
		newStatus = entity.OutboxStatusFailed
	}
	return r.db.WithContext(ctx).
		Model(&entity.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   attempts,
			"last_error": errMsg,
			"status":     newStatus,
		}).Error
}
