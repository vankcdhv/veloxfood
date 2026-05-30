package repository

import (
	"context"

	"project/services/store/internal/entity"

	"gorm.io/gorm"
)

// OutboxRepository manages transactional outbox events for reliable Kafka publishing.
// Append is called inside a business TX; Pickup/MarkPublished/MarkFailed are used by the dispatcher.
type OutboxRepository interface {
	// Append inserts an OutboxEvent using the provided transaction.
	// tx must be a *gorm.DB already inside a transaction.
	Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error

	// Pickup returns up to limit pending events with fewer than maxAttempts retries,
	// ordered oldest-first.
	Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error)

	// MarkPublished marks an event as published (status='published', published_at=now).
	MarkPublished(ctx context.Context, id string) error

	// MarkFailed increments attempts and records the error message.
	// Sets status='failed' when attempts >= maxAttempts.
	MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}
