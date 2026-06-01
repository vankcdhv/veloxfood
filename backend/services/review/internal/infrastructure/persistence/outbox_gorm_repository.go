package persistence

import (
	"context"
	"encoding/json"
	"time"

	"project/pkg/outbox"
	"project/services/review/internal/entity"
	"project/services/review/internal/repository"

	"gorm.io/gorm"
)

// ── OutboxEvent ──────────────────────────────────────────────────────────────

type outboxGormRepository struct{ db *gorm.DB }

func NewOutboxGormRepository(db *gorm.DB) repository.OutboxRepository {
	return &outboxGormRepository{db: db}
}

func (r *outboxGormRepository) Insert(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error {
	return tx.WithContext(ctx).Create(evt).Error
}

func (r *outboxGormRepository) Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error) {
	var rows []*entity.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = 'pending' AND attempts < ?", maxAttempts).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *outboxGormRepository) MarkPublished(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.OutboxEvent{}).
		Where("id = ? AND status = 'pending'", id).
		Updates(map[string]any{"status": "published", "published_at": now}).Error
}

func (r *outboxGormRepository) MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error {
	newStatus := "pending"
	if attempts >= maxAttempts {
		newStatus = "failed"
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

// outboxDispatchAdapter bridges repository.OutboxRepository to pkg/outbox.DispatchRepository.
type outboxDispatchAdapter struct{ repo repository.OutboxRepository }

// NewOutboxDispatchAdapter wraps the review outbox repo for pkg/outbox.Dispatcher.
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
		result[i] = &outbox.OutboxRow{
			ID:            r.ID,
			AggregateType: r.AggregateType,
			AggregateID:   r.AggregateID,
			EventType:     r.EventType,
			Payload:       json.RawMessage(r.Payload),
			TraceID:       r.TraceID,
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

// ── ProcessedEvent ───────────────────────────────────────────────────────────

type processedEventGormRepository struct{ db *gorm.DB }

func NewProcessedEventGormRepository(db *gorm.DB) repository.ProcessedEventRepository {
	return &processedEventGormRepository{db: db}
}

func (r *processedEventGormRepository) Insert(ctx context.Context, tx *gorm.DB, eventID string) error {
	return tx.WithContext(ctx).Create(&entity.ProcessedEvent{EventID: eventID}).Error
}
