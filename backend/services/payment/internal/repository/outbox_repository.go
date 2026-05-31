package repository

import (
	"context"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// OutboxRepository manages transactional outbox events for reliable Kafka publishing.
// Append is called inside a business TX; Pickup/MarkPublished/MarkFailed are used by the dispatcher.
type OutboxRepository interface {
	// Append inserts an OutboxEvent using the provided transaction.
	Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error

	// Pickup returns up to limit pending events with fewer than maxAttempts retries.
	Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error)

	// MarkPublished marks an event as published.
	MarkPublished(ctx context.Context, id string) error

	// MarkFailed increments attempts and records the error; sets status=failed when maxAttempts reached.
	MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}
