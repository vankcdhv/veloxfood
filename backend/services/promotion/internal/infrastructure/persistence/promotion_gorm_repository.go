package persistence

import (
	"context"
	"errors"
	"time"

	"project/services/promotion/internal/entity"
	"project/services/promotion/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type promotionGormRepository struct {
	db *gorm.DB
}

// NewPromotionGormRepository returns a repository.PromotionRepository backed by GORM.
func NewPromotionGormRepository(db *gorm.DB) repository.PromotionRepository {
	return &promotionGormRepository{db: db}
}

func (r *promotionGormRepository) Create(ctx context.Context, p *entity.Promotion) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *promotionGormRepository) GetByID(ctx context.Context, id string) (*entity.Promotion, error) {
	var p entity.Promotion
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrPromotionNotFound
		}
		return nil, err
	}
	return &p, nil
}

// GetByStoreAndCodeForUpdate loads the promotion row with SELECT … FOR UPDATE so
// concurrent ApplyPromotion calls serialise on the same voucher. Must be called
// within an open transaction passed as tx.
func (r *promotionGormRepository) GetByStoreAndCodeForUpdate(ctx context.Context, tx *gorm.DB, storeID, code string) (*entity.Promotion, error) {
	var p entity.Promotion
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("store_id = ? AND code = ? AND deleted_at IS NULL", storeID, code).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrPromotionNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *promotionGormRepository) ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.Promotion, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.Promotion{}).
		Where("store_id = ?", storeID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*entity.Promotion
	err := r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *promotionGormRepository) Update(ctx context.Context, p *entity.Promotion) error {
	return r.db.WithContext(ctx).Model(p).Updates(map[string]any{
		"value":        p.Value,
		"min_order":    p.MinOrder,
		"max_discount": p.MaxDiscount,
		"starts_at":    p.StartsAt,
		"ends_at":      p.EndsAt,
		"usage_limit":  p.UsageLimit,
		"status":       p.Status,
		"updated_at":   p.UpdatedAt,
	}).Error
}

// IncrementUsedCount adds 1 to used_count atomically inside the provided tx.
func (r *promotionGormRepository) IncrementUsedCount(ctx context.Context, tx *gorm.DB, promotionID string) error {
	return tx.WithContext(ctx).
		Model(&entity.Promotion{}).
		Where("id = ?", promotionID).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// DecrementUsedCount subtracts 1 from used_count atomically, floored at zero.
func (r *promotionGormRepository) DecrementUsedCount(ctx context.Context, tx *gorm.DB, promotionID string) error {
	return tx.WithContext(ctx).
		Model(&entity.Promotion{}).
		Where("id = ? AND used_count > 0", promotionID).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error
}

func (r *promotionGormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Promotion{}, "id = ?", id).Error
}

// ─── PromotionUsage ──────────────────────────────────────────────────────────

type promotionUsageGormRepository struct {
	db *gorm.DB
}

// NewPromotionUsageGormRepository returns a repository.PromotionUsageRepository backed by GORM.
func NewPromotionUsageGormRepository(db *gorm.DB) repository.PromotionUsageRepository {
	return &promotionUsageGormRepository{db: db}
}

func (r *promotionUsageGormRepository) Create(ctx context.Context, tx *gorm.DB, u *entity.PromotionUsage) error {
	return tx.WithContext(ctx).Create(u).Error
}

func (r *promotionUsageGormRepository) ListByOrderID(ctx context.Context, orderID string) ([]*entity.PromotionUsage, error) {
	var rows []*entity.PromotionUsage
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Find(&rows).Error
	return rows, err
}

func (r *promotionUsageGormRepository) ConfirmByOrderID(ctx context.Context, tx *gorm.DB, orderID string) error {
	return tx.WithContext(ctx).
		Model(&entity.PromotionUsage{}).
		Where("order_id = ? AND status = ?", orderID, "RESERVED").
		Updates(map[string]any{"status": "CONFIRMED"}).Error
}

// VoidByOrderID transitions an order's RESERVED **and CONFIRMED** usages to
// VOIDED and returns the affected rows so the caller can decrement used_count.
// CONFIRMED must be releasable too: cancelling or rejecting a placed order
// (voucher already confirmed) has to return the quota — otherwise every
// cancelled order leaks one usage forever. Only order-scoped callers reach
// this (cancel saga/consumer, placement compensation); the TTL janitor sweeps
// by expired RESERVED rows and never touches live orders' CONFIRMED usages.
func (r *promotionUsageGormRepository) VoidByOrderID(ctx context.Context, tx *gorm.DB, orderID string) ([]*entity.PromotionUsage, error) {
	var rows []*entity.PromotionUsage
	if err := tx.WithContext(ctx).
		Where("order_id = ? AND status IN ?", orderID, []string{"RESERVED", "CONFIRMED"}).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if err := tx.WithContext(ctx).
		Model(&entity.PromotionUsage{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"status": "VOIDED"}).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ExpiredReservedOrderIDs returns distinct order ids holding RESERVED usages
// created before the cutoff. These are orphaned reservations — the placing
// saga neither confirmed nor released them (e.g. crash mid-compensation) —
// and the janitor releases them so used_count stops over-counting.
func (r *promotionUsageGormRepository) ExpiredReservedOrderIDs(ctx context.Context, before time.Time, limit int) ([]string, error) {
	ids := []string{}
	err := r.db.WithContext(ctx).
		Model(&entity.PromotionUsage{}).
		Distinct("order_id").
		Where("status = ? AND created_at < ?", "RESERVED", before).
		Limit(limit).
		Pluck("order_id", &ids).Error
	return ids, err
}
