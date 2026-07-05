package repository

import (
	"context"
	"time"

	"project/services/promotion/internal/entity"

	"gorm.io/gorm"
)

// PromotionRepository manages the Promotion aggregate.
type PromotionRepository interface {
	// Create persists a new promotion.
	Create(ctx context.Context, p *entity.Promotion) error

	// GetByID returns the promotion with the given ID, or gorm.ErrRecordNotFound.
	GetByID(ctx context.Context, id string) (*entity.Promotion, error)

	// GetByStoreAndCodeForUpdate loads a promotion by (store_id, code) with a
	// SELECT … FOR UPDATE lock. Must be called inside a transaction (tx).
	GetByStoreAndCodeForUpdate(ctx context.Context, tx *gorm.DB, storeID, code string) (*entity.Promotion, error)

	// ListByStore returns a page of non-deleted promotions for a store (newest
	// first) plus the total count.
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.Promotion, int64, error)

	// Update saves mutable fields (value, min_order, max_discount, starts_at,
	// ends_at, usage_limit, status) on an existing promotion.
	Update(ctx context.Context, p *entity.Promotion) error

	// IncrementUsedCount atomically increments used_count inside a transaction.
	IncrementUsedCount(ctx context.Context, tx *gorm.DB, promotionID string) error

	// DecrementUsedCount atomically decrements used_count inside a transaction.
	DecrementUsedCount(ctx context.Context, tx *gorm.DB, promotionID string) error

	// Delete soft-deletes the promotion.
	Delete(ctx context.Context, id string) error
}

// PromotionUsageRepository manages usage reservation records.
type PromotionUsageRepository interface {
	// Create inserts a new usage row (status=RESERVED) inside a transaction.
	Create(ctx context.Context, tx *gorm.DB, u *entity.PromotionUsage) error

	// ListByOrderID returns all usage rows for a given order.
	ListByOrderID(ctx context.Context, orderID string) ([]*entity.PromotionUsage, error)

	// ConfirmByOrderID transitions all RESERVED usages for an order to CONFIRMED
	// within tx. Idempotent — already-CONFIRMED rows are untouched.
	ConfirmByOrderID(ctx context.Context, tx *gorm.DB, orderID string) error

	// VoidByOrderID transitions all RESERVED usages for an order to VOIDED.
	// Returns the voided rows so the caller can decrement used_count.
	// Idempotent — rows already VOIDED or CONFIRMED are untouched.
	VoidByOrderID(ctx context.Context, tx *gorm.DB, orderID string) ([]*entity.PromotionUsage, error)

	// ExpiredReservedOrderIDs returns distinct order ids that still hold
	// RESERVED usages created before the cutoff — reservations whose saga
	// never confirmed nor released them (crash mid-compensation).
	ExpiredReservedOrderIDs(ctx context.Context, before time.Time, limit int) ([]string, error)
}
