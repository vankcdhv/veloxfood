package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"project/services/order/internal/entity"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type orderGormRepository struct {
	db *gorm.DB
}

// NewOrderGormRepository returns an OrderRepository backed by GORM/Postgres.
func NewOrderGormRepository(db *gorm.DB) repository.OrderRepository {
	return &orderGormRepository{db: db}
}

func (r *orderGormRepository) NextDailyCodeSeq(ctx context.Context, day time.Time) (int, error) {
	// Atomic upsert: first order of the day starts at 1, subsequent ones increment.
	var seq int
	err := r.db.WithContext(ctx).Raw(
		`INSERT INTO order_code_seq (day, seq) VALUES (?, 1)
		 ON CONFLICT (day) DO UPDATE SET seq = order_code_seq.seq + 1
		 RETURNING seq`,
		day.Format("2006-01-02"),
	).Scan(&seq).Error
	if err != nil {
		return 0, fmt.Errorf("next daily code seq: %w", err)
	}
	return seq, nil
}

func (r *orderGormRepository) Create(ctx context.Context, tx *gorm.DB, order *entity.Order, items []*entity.OrderItem) error {
	if err := tx.WithContext(ctx).Create(order).Error; err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	for _, item := range items {
		item.OrderID = order.ID
		if err := tx.WithContext(ctx).Create(item).Error; err != nil {
			return fmt.Errorf("create order item: %w", err)
		}
	}
	return nil
}

func (r *orderGormRepository) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	var order entity.Order
	err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return &order, nil
}

func (r *orderGormRepository) GetByIDForUpdate(ctx context.Context, tx *gorm.DB, id string) (*entity.Order, error) {
	var order entity.Order
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get order for update: %w", err)
	}
	// Load items separately (preload + FOR UPDATE doesn't compose cleanly).
	if err := tx.WithContext(ctx).Where("order_id = ?", id).Find(&order.Items).Error; err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	return &order, nil
}

func (r *orderGormRepository) ListByCustomer(ctx context.Context, customerID string, page, pageSize int) ([]*entity.Order, int64, error) {
	var orders []*entity.Order
	var total int64
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Model(&entity.Order{}).Where("customer_id = ?", customerID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("placed_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&orders).Error
	return orders, total, err
}

func (r *orderGormRepository) ListByStore(ctx context.Context, storeID string, status entity.OrderStatus, page, pageSize int) ([]*entity.Order, int64, error) {
	var orders []*entity.Order
	var total int64
	offset := (page - 1) * pageSize
	q := r.db.WithContext(ctx).Where("store_id = ?", storeID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Model(&entity.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Items").Order("placed_at DESC").Limit(pageSize).Offset(offset).Find(&orders).Error
	return orders, total, err
}

func (r *orderGormRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.OrderStatus, changedBy *string, note *string) error {
	if err := tx.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", orderID).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	hist := &entity.OrderStatusHistory{
		OrderID:   orderID,
		Status:    status,
		ChangedBy: changedBy,
		Note:      note,
	}
	return tx.WithContext(ctx).Create(hist).Error
}

func (r *orderGormRepository) UpdatePaymentStatus(ctx context.Context, tx *gorm.DB, orderID string, status entity.PaymentStatus) error {
	return tx.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", orderID).
		Updates(map[string]any{"payment_status": status, "updated_at": time.Now()}).Error
}

func (r *orderGormRepository) StatusChangedAt(ctx context.Context, orderID string, status entity.OrderStatus) (*time.Time, error) {
	var hist entity.OrderStatusHistory
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND status = ?", orderID, status).
		Order("created_at DESC").
		First(&hist).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("status changed-at: %w", err)
	}
	return &hist.CreatedAt, nil
}

