package persistence

import (
	"context"

	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
)

type incidentGormRepository struct {
	db *gorm.DB
}

// NewIncidentGormRepository returns an IncidentRepository backed by GORM.
func NewIncidentGormRepository(db *gorm.DB) repository.IncidentRepository {
	return &incidentGormRepository{db: db}
}

func (r *incidentGormRepository) Create(ctx context.Context, tx *gorm.DB, inc *entity.DeliveryIncident) error {
	return tx.WithContext(ctx).Create(inc).Error
}

func (r *incidentGormRepository) ListAll(ctx context.Context) ([]*entity.DeliveryIncident, error) {
	var rows []*entity.DeliveryIncident
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *incidentGormRepository) ListAllWithOrderCode(ctx context.Context) ([]*repository.IncidentWithOrder, error) {
	var rows []*repository.IncidentWithOrder
	err := r.db.WithContext(ctx).
		Table("delivery_incidents").
		Select("delivery_incidents.*, deliveries.order_code AS order_code").
		Joins("LEFT JOIN deliveries ON deliveries.id = delivery_incidents.delivery_id").
		Order("delivery_incidents.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *incidentGormRepository) UpdateStatus(ctx context.Context, id string, status entity.IncidentStatus) error {
	res := r.db.WithContext(ctx).
		Model(&entity.DeliveryIncident{}).
		Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
