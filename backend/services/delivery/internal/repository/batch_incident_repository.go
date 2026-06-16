package repository

import (
	"context"

	"project/services/delivery/internal/entity"

	"gorm.io/gorm"
)

// BatchRepository is the persistence contract for delivery batches.
type BatchRepository interface {
	// FindOpenBatch returns the shipper's OPEN batch for the given store, if any.
	FindOpenBatch(ctx context.Context, tx *gorm.DB, shipperID, storeID string) (*entity.DeliveryBatch, error)

	// CreateBatch inserts a new OPEN batch inside the given transaction.
	CreateBatch(ctx context.Context, tx *gorm.DB, b *entity.DeliveryBatch) error
}

// IncidentWithOrder is an incident row enriched with the human-readable order
// code from its delivery, so the admin view never exposes a raw order UUID.
type IncidentWithOrder struct {
	entity.DeliveryIncident
	OrderCode string `gorm:"column:order_code"`
}

// IncidentRepository is the persistence contract for delivery incidents.
type IncidentRepository interface {
	// Create inserts a new incident record.
	Create(ctx context.Context, tx *gorm.DB, inc *entity.DeliveryIncident) error

	// ListAll returns all incidents (admin view), newest first.
	ListAll(ctx context.Context) ([]*entity.DeliveryIncident, error)

	// ListAllWithOrderCode returns a page of incidents joined to their delivery's
	// order_code (admin view, newest first) plus the total count.
	ListAllWithOrderCode(ctx context.Context, limit, offset int) ([]*IncidentWithOrder, int64, error)

	// UpdateStatus sets an incident's status by id. Returns ErrNotFound when no
	// row matches.
	UpdateStatus(ctx context.Context, id string, status entity.IncidentStatus) error
}

// OutboxRepository persists outbox events for the delivery service.
type OutboxRepository interface {
	Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error
	Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}

// ProcessedEventRepository deduplicates incoming Kafka events.
type ProcessedEventRepository interface {
	// MarkProcessed inserts event_id. Returns (true, nil) on first insert,
	// (false, nil) on duplicate (already processed), or (false, err) on failure.
	MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (bool, error)
}
