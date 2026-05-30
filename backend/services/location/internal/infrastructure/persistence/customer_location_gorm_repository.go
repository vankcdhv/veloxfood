package persistence

import (
	"context"

	"project/services/location/internal/entity"
	"project/services/location/internal/repository"

	"gorm.io/gorm"
)

type customerLocationGormRepository struct {
	db *gorm.DB
}

func NewCustomerLocationGormRepository(db *gorm.DB) repository.CustomerLocationRepository {
	return &customerLocationGormRepository{db: db}
}

func (r *customerLocationGormRepository) Create(ctx context.Context, l *entity.CustomerLocation) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *customerLocationGormRepository) GetByID(ctx context.Context, id string) (*entity.CustomerLocation, error) {
	var l entity.CustomerLocation
	if err := r.db.WithContext(ctx).First(&l, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *customerLocationGormRepository) ListByCustomer(ctx context.Context, customerID string) ([]*entity.CustomerLocation, error) {
	var rows []*entity.CustomerLocation
	return rows, r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("is_default DESC, created_at DESC").
		Find(&rows).Error
}

func (r *customerLocationGormRepository) Delete(ctx context.Context, id, customerID string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", id, customerID).
		Delete(&entity.CustomerLocation{}).Error
}

// SetDefault clears the customer's current default then sets the target, in one TX
// (the partial unique index allows only one default per customer).
func (r *customerLocationGormRepository) SetDefault(ctx context.Context, id, customerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.CustomerLocation{}).
			Where("customer_id = ? AND is_default = ?", customerID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&entity.CustomerLocation{}).
			Where("id = ? AND customer_id = ?", id, customerID).
			Update("is_default", true).Error
	})
}
