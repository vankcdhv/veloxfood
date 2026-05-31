package repository

import (
	"context"

	"project/services/payment/internal/entity"

	"gorm.io/gorm"
)

// PaymentRepository manages payment records.
type PaymentRepository interface {
	// Create inserts a new payment within tx.
	Create(ctx context.Context, tx *gorm.DB, p *entity.Payment) error

	// GetByOrderID returns the first payment for an order, or nil if not found.
	GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)

	// GetByIdempotencyKey returns the payment matching the key, or nil if absent.
	GetByIdempotencyKey(ctx context.Context, key string) (*entity.Payment, error)

	// UpdateStatus changes the payment status (and optional momo_trans_id) within tx.
	UpdateStatus(ctx context.Context, tx *gorm.DB, paymentID string, status entity.PaymentStatus, momoTransID *string) error
}
