package repository

import (
	"context"

	"project/services/delivery/internal/entity"

	"gorm.io/gorm"
)

// DeliveryRepository is the persistence contract for deliveries.
type DeliveryRepository interface {
	// Create inserts a new delivery record inside the given transaction.
	Create(ctx context.Context, tx *gorm.DB, d *entity.Delivery) error

	// GetByOrderID loads a delivery by order_id.
	GetByOrderID(ctx context.Context, orderID string) (*entity.Delivery, error)

	// GetByOrderIDForUpdate loads with SELECT … FOR UPDATE inside a transaction.
	GetByOrderIDForUpdate(ctx context.Context, tx *gorm.DB, orderID string) (*entity.Delivery, error)

	// ListAvailable returns all AVAILABLE deliveries, ordered by created_at ASC.
	ListAvailable(ctx context.Context) ([]*entity.Delivery, error)

	// ListByShipper returns all deliveries claimed by the given shipper.
	ListByShipper(ctx context.Context, shipperID string) ([]*entity.Delivery, error)

	// UpdateStatus sets delivery status and optional timestamps inside a transaction.
	UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.DeliveryStatus, shipperID, batchID *string) error

	// UpdateStatusByOrderID atomically claims: sets shipper_id, batch_id, status, claimed_at
	// WHERE order_id=? AND shipper_id IS NULL AND status='AVAILABLE'.
	// Returns RowsAffected so callers can detect concurrent claim races.
	ClaimDelivery(ctx context.Context, tx *gorm.DB, orderID, shipperID, batchID string) (int64, error)

	// MarkDelivered sets status=DELIVERED and delivered_at=now() inside tx.
	MarkDelivered(ctx context.Context, tx *gorm.DB, orderID string) error

	// SetStoreDelivering moves the delivery to STORE_DELIVERING.
	SetStoreDelivering(ctx context.Context, tx *gorm.DB, orderID string) error

	// Cancel moves the delivery to CANCELLED.
	Cancel(ctx context.Context, tx *gorm.DB, orderID string) error

	// CountBatchActiveDeliveries returns the count of deliveries in active statuses
	// for the given batch_id. Used to enforce the per-batch max size.
	CountBatchActiveDeliveries(ctx context.Context, tx *gorm.DB, batchID string) (int64, error)

	// ListAll returns all deliveries (admin view), newest first.
	ListAll(ctx context.Context) ([]*entity.Delivery, error)
}
