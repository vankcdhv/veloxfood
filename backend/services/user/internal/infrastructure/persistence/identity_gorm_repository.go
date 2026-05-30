package persistence

import (
	"context"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type identityGormRepository struct {
	db *gorm.DB
}

func NewIdentityGormRepository(db *gorm.DB) repository.IdentityRepository {
	return &identityGormRepository{db: db}
}

func (r *identityGormRepository) GetByProviderSubject(ctx context.Context, provider entity.IdentityProvider, subject string) (*entity.Identity, error) {
	var id entity.Identity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND external_subject = ?", provider, subject).
		First(&id).Error
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *identityGormRepository) Create(ctx context.Context, identity *entity.Identity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}
