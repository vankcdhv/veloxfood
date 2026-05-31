package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type deliveryGormRepository struct {
	db *gorm.DB
}

// NewDeliveryGormRepository returns a DeliveryRepository backed by GORM/Postgres.
func NewDeliveryGormRepository(db *gorm.DB) repository.DeliveryRepository {
	return &deliveryGormRepository{db: db}
}

func (r *deliveryGormRepository) Create(ctx context.Context, tx *gorm.DB, d *entity.Delivery) error {
	return tx.WithContext(ctx).Create(d).Error
}

func (r *deliveryGormRepository) GetByOrderID(ctx context.Context, orderID string) (*entity.Delivery, error) {
	var d entity.Delivery
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get delivery by order_id: %w", err)
	}
	return &d, nil
}

func (r *deliveryGormRepository) GetByOrderIDForUpdate(ctx context.Context, tx *gorm.DB, orderID string) (*entity.Delivery, error) {
	var d entity.Delivery
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ?", orderID).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get delivery for update: %w", err)
	}
	return &d, nil
}

func (r *deliveryGormRepository) ListAvailable(ctx context.Context) ([]*entity.Delivery, error) {
	var rows []*entity.Delivery
	err := r.db.WithContext(ctx).
		Where("status = ?", entity.DeliveryAvailable).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *deliveryGormRepository) ListByShipper(ctx context.Context, shipperID string) ([]*entity.Delivery, error) {
	var rows []*entity.Delivery
	err := r.db.WithContext(ctx).
		Where("shipper_id = ?", shipperID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *deliveryGormRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.DeliveryStatus, shipperID, batchID *string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if shipperID != nil {
		updates["shipper_id"] = *shipperID
	}
	if batchID != nil {
		updates["batch_id"] = *batchID
	}
	return tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("order_id = ?", orderID).
		Updates(updates).Error
}

// ClaimDelivery atomically claims the delivery: sets shipper_id, batch_id, status=CLAIMED,
// claimed_at=now() only when the row is still AVAILABLE with no shipper (prevents races).
// Returns RowsAffected — callers MUST check for 0 to detect a lost race.
func (r *deliveryGormRepository) ClaimDelivery(ctx context.Context, tx *gorm.DB, orderID, shipperID, batchID string) (int64, error) {
	now := time.Now()
	result := tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("order_id = ? AND shipper_id IS NULL AND status = ?", orderID, entity.DeliveryAvailable).
		Updates(map[string]any{
			"shipper_id":  shipperID,
			"batch_id":    batchID,
			"status":      entity.DeliveryClaimed,
			"claimed_at":  now,
			"updated_at":  now,
		})
	return result.RowsAffected, result.Error
}

func (r *deliveryGormRepository) MarkDelivered(ctx context.Context, tx *gorm.DB, orderID string) error {
	now := time.Now()
	return tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("order_id = ?", orderID).
		Updates(map[string]any{
			"status":       entity.DeliveryDelivered,
			"delivered_at": now,
			"updated_at":   now,
		}).Error
}

func (r *deliveryGormRepository) SetStoreDelivering(ctx context.Context, tx *gorm.DB, orderID string) error {
	return tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("order_id = ? AND status = ?", orderID, entity.DeliveryAvailable).
		Updates(map[string]any{
			"status":     entity.DeliveryStoreDelivering,
			"updated_at": time.Now(),
		}).Error
}

func (r *deliveryGormRepository) Cancel(ctx context.Context, tx *gorm.DB, orderID string) error {
	return tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("order_id = ?", orderID).
		Updates(map[string]any{
			"status":     entity.DeliveryCancelled,
			"updated_at": time.Now(),
		}).Error
}

// CountBatchActiveDeliveries counts deliveries in active (non-terminal) statuses
// for the given batch. Active = CLAIMED, PICKED_UP, DELIVERING.
func (r *deliveryGormRepository) CountBatchActiveDeliveries(ctx context.Context, tx *gorm.DB, batchID string) (int64, error) {
	var count int64
	activeStatuses := []entity.DeliveryStatus{
		entity.DeliveryClaimed,
		entity.DeliveryPickedUp,
		entity.DeliveryDelivering,
	}
	err := tx.WithContext(ctx).
		Model(&entity.Delivery{}).
		Where("batch_id = ? AND status IN ?", batchID, activeStatuses).
		Count(&count).Error
	return count, err
}

func (r *deliveryGormRepository) ListAll(ctx context.Context) ([]*entity.Delivery, error) {
	var rows []*entity.Delivery
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}
