package persistence

import (
	"context"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

type storeGormRepository struct {
	db *gorm.DB
}

// NewStoreGormRepository returns a repository.StoreRepository backed by GORM.
func NewStoreGormRepository(db *gorm.DB) repository.StoreRepository {
	return &storeGormRepository{db: db}
}

func (r *storeGormRepository) Create(ctx context.Context, s *entity.Store) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *storeGormRepository) GetByID(ctx context.Context, id string) (*entity.Store, error) {
	var s entity.Store
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *storeGormRepository) GetByVendorID(ctx context.Context, vendorID string) (*entity.Store, error) {
	var s entity.Store
	if err := r.db.WithContext(ctx).First(&s, "vendor_id = ?", vendorID).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *storeGormRepository) List(ctx context.Context, saleStatus string) ([]*entity.Store, error) {
	q := r.db.WithContext(ctx).Order("name ASC")
	if saleStatus != "" {
		q = q.Where("sale_status = ?", saleStatus)
	}
	var rows []*entity.Store
	return rows, q.Find(&rows).Error
}

func (r *storeGormRepository) ListByOwner(ctx context.Context, ownerUserID string) ([]*entity.Store, error) {
	var rows []*entity.Store
	return rows, r.db.WithContext(ctx).
		Where("owner_user_id = ?", ownerUserID).
		Order("name ASC").
		Find(&rows).Error
}

func (r *storeGormRepository) Update(ctx context.Context, s *entity.Store) error {
	return r.db.WithContext(ctx).Model(s).Updates(map[string]any{
		"name":           s.Name,
		"business_type":  s.BusinessType,
		"address":        s.Address,
		"phone":          s.Phone,
		"sale_status":    s.SaleStatus,
		"pickup_enabled": s.PickupEnabled,
		"prep_minutes":   s.PrepMinutes,
	}).Error
}

func (r *storeGormRepository) UpdateSaleStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Store{}).
		Where("id = ?", id).
		Update("sale_status", status).Error
}

func (r *storeGormRepository) UpdatePickupEnabled(ctx context.Context, id string, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&entity.Store{}).
		Where("id = ?", id).
		Update("pickup_enabled", enabled).Error
}

func (r *storeGormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Store{}, "id = ?", id).Error
}
