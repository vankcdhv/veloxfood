package persistence

import (
	"context"
	"errors"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type permissionGormRepository struct {
	db *gorm.DB
}

// NewPermissionGormRepository returns a PermissionRepository backed by GORM.
func NewPermissionGormRepository(db *gorm.DB) repository.PermissionRepository {
	return &permissionGormRepository{db: db}
}

func (r *permissionGormRepository) List(ctx context.Context, filter repository.PermissionFilter) ([]*entity.Permission, error) {
	q := r.db.WithContext(ctx).Model(&entity.Permission{})
	if filter.Resource != nil {
		q = q.Where("resource = ?", *filter.Resource)
	}
	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}
	if filter.IsSystem != nil {
		q = q.Where("is_system = ?", *filter.IsSystem)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	q = q.Order("code ASC").Limit(limit).Offset(filter.Offset)

	var perms []*entity.Permission
	if err := q.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *permissionGormRepository) GetByID(ctx context.Context, id string) (*entity.Permission, error) {
	var perm entity.Permission
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&perm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &perm, nil
}

func (r *permissionGormRepository) GetByCode(ctx context.Context, code string) (*entity.Permission, error) {
	var perm entity.Permission
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&perm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &perm, nil
}

func (r *permissionGormRepository) Create(ctx context.Context, perm *entity.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *permissionGormRepository) Update(ctx context.Context, perm *entity.Permission) error {
	return r.db.WithContext(ctx).Save(perm).Error
}

func (r *permissionGormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.Permission{}).Error
}
