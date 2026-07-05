package repository

import (
	"context"

	"project/services/order/internal/entity"
)

// CompensationRepository persists durable saga-rollback intents.
type CompensationRepository interface {
	// Create records a compensation that could not be executed in-request.
	Create(ctx context.Context, c *entity.PendingCompensation) error
	// ListOpen returns undone rows with attempts below maxAttempts, oldest first.
	ListOpen(ctx context.Context, maxAttempts, limit int) ([]*entity.PendingCompensation, error)
	// MarkDone stamps done_at.
	MarkDone(ctx context.Context, id string) error
	// MarkFailed increments attempts and records the error.
	MarkFailed(ctx context.Context, id, errMsg string) error
}
