package persistence

import (
	"context"
	"errors"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
)

type paymentGormRepository struct {
	db *gorm.DB
}

func NewPaymentGormRepository(db *gorm.DB) repository.PaymentRepository {
	return &paymentGormRepository{db: db}
}

func (r *paymentGormRepository) Create(ctx context.Context, tx *gorm.DB, p *entity.Payment) error {
	return tx.WithContext(ctx).Create(p).Error
}

func (r *paymentGormRepository) GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error) {
	var p entity.Payment
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *paymentGormRepository) GetByIdempotencyKey(ctx context.Context, key string) (*entity.Payment, error) {
	var p entity.Payment
	err := r.db.WithContext(ctx).
		Where("idempotency_key = ?", key).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *paymentGormRepository) UpdateStatus(
	ctx context.Context, tx *gorm.DB,
	paymentID string, status entity.PaymentStatus, momoTransID *string,
) error {
	updates := map[string]any{"status": status}
	if momoTransID != nil {
		updates["momo_trans_id"] = *momoTransID
	}
	return tx.WithContext(ctx).
		Model(&entity.Payment{}).
		Where("id = ?", paymentID).
		Updates(updates).Error
}
