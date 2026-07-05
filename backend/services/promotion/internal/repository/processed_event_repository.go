package repository

import (
	"context"

	"gorm.io/gorm"
)

// ProcessedEventRepository tracks consumed Kafka event IDs for idempotency.
type ProcessedEventRepository interface {
	// MarkProcessed inserts event_id within tx. Returns false (no error) if
	// already present (duplicate delivery), true if newly inserted.
	MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (inserted bool, err error)
}
