package persistence

import (
	"context"
	"time"

	"project/pkg/outbox"
	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

type outboxGormRepository struct {
	db *gorm.DB
}

// NewOutboxGormRepository returns an OutboxRepository backed by GORM.
func NewOutboxGormRepository(db *gorm.DB) repository.OutboxRepository {
	return &outboxGormRepository{db: db}
}

func (r *outboxGormRepository) Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error {
	return tx.WithContext(ctx).Create(evt).Error
}

func (r *outboxGormRepository) Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error) {
	var events []*entity.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ? AND attempts < ?", entity.OutboxStatusPending, maxAttempts).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

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

// outboxDispatchAdapter bridges repository.OutboxRepository to pkg/outbox.DispatchRepository
// so the pkg/outbox.Dispatcher can poll this service's outbox table without importing
// any service-level package.
type outboxDispatchAdapter struct {
	repo repository.OutboxRepository
}

// NewOutboxDispatchAdapter wraps the store outbox repo for use by pkg/outbox.Dispatcher.
func NewOutboxDispatchAdapter(repo repository.OutboxRepository) outbox.DispatchRepository {
	return &outboxDispatchAdapter{repo: repo}
}

func (a *outboxDispatchAdapter) Pickup(ctx context.Context, limit, maxAttempts int) ([]*outbox.OutboxRow, error) {
	rows, err := a.repo.Pickup(ctx, limit, maxAttempts)
	if err != nil {
		return nil, err
	}
	result := make([]*outbox.OutboxRow, len(rows))
	for i, r := range rows {
		traceID := ""
		if r.TraceID != nil {
			traceID = *r.TraceID
		}
		result[i] = &outbox.OutboxRow{
			ID:            r.ID,
			AggregateType: r.AggregateType,
			AggregateID:   r.AggregateID,
			EventType:     r.EventType,
			Payload:       r.Payload,
			TraceID:       traceID,
			Attempts:      r.Attempts,
		}
	}
	return result, nil
}

func (a *outboxDispatchAdapter) MarkPublished(ctx context.Context, id string) error {
	return a.repo.MarkPublished(ctx, id)
}

func (a *outboxDispatchAdapter) MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error {
	return a.repo.MarkFailed(ctx, id, errMsg, attempts, maxAttempts)
}
