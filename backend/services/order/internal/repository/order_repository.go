package repository

import (
	"context"
	"time"

	"project/services/order/internal/entity"

	"gorm.io/gorm"
)

// OrderRepository is the persistence contract for orders and their sub-aggregates.
type OrderRepository interface {
	// Create inserts an order + its items inside the given transaction.
	Create(ctx context.Context, tx *gorm.DB, order *entity.Order, items []*entity.OrderItem) error

	// NextDailyCodeSeq atomically returns the next per-day order sequence number
	// (1-based), used to build human-readable codes like VLX-260601-001.
	NextDailyCodeSeq(ctx context.Context, day time.Time) (int, error)

	// GetByID loads an order with its items. Returns gorm.ErrRecordNotFound when absent.
	GetByID(ctx context.Context, id string) (*entity.Order, error)

	// GetByIDForUpdate loads an order with a SELECT … FOR UPDATE lock.
	GetByIDForUpdate(ctx context.Context, tx *gorm.DB, id string) (*entity.Order, error)

	// ListByCustomer returns paginated orders for a customer, newest first.
	ListByCustomer(ctx context.Context, customerID string, page, pageSize int) ([]*entity.Order, int64, error)

	// ListByStore returns paginated orders for a store with optional status filter.
	ListByStore(ctx context.Context, storeID string, status entity.OrderStatus, page, pageSize int) ([]*entity.Order, int64, error)

	// UpdateStatus sets order.status and appends a status history row.
	// Both writes happen inside tx to keep them atomic.
	UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.OrderStatus, changedBy *string, note *string) error

	// UpdatePaymentStatus sets order.payment_status.
	UpdatePaymentStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.PaymentStatus) error
}

// OutboxRepository persists outbox events for the order service.
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
