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
