package repository

import (
	"context"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// PayoutRepository manages payout batch records.
type PayoutRepository interface {
	// Create inserts a new payout batch.
	Create(ctx context.Context, p *entity.PayoutBatch) error

	// GetByID returns a payout batch by ID.
	GetByID(ctx context.Context, id string) (*entity.PayoutBatch, error)

	// ListByStore returns payout batches for a store (newest first).
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.PayoutBatch, error)

	// MarkSettled atomically marks the batch SETTLED within tx.
	MarkSettled(ctx context.Context, tx *gorm.DB, batchID, settledByUserID string) error
}
