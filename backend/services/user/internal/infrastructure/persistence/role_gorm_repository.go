package persistence

import (
	"context"
	"errors"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type roleGormRepository struct {
	db *gorm.DB
}

// NewRoleGormRepository returns a RoleRepository backed by GORM.
func NewRoleGormRepository(db *gorm.DB) repository.RoleRepository {
	return &roleGormRepository{db: db}
}

func (r *roleGormRepository) List(ctx context.Context, filter repository.RoleFilter) ([]*entity.Role, error) {
	q := r.db.WithContext(ctx).Model(&entity.Role{})
	if filter.ScopeType != nil {
		q = q.Where("scope_type = ?", *filter.ScopeType)
	}
	if filter.IsSystem != nil {
		q = q.Where("is_system = ?", *filter.IsSystem)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	q = q.Order("created_at DESC").Limit(limit).Offset(filter.Offset)

	var roles []*entity.Role
	if err := q.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *roleGormRepository) GetByID(ctx context.Context, id string) (*entity.Role, error) {
	var role entity.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleGormRepository) GetByCode(ctx context.Context, code string) (*entity.Role, error) {
	var role entity.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleGormRepository) Create(ctx context.Context, role *entity.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleGormRepository) Update(ctx context.Context, role *entity.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleGormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.Role{}).Error
}

func (r *roleGormRepository) ListPermissions(ctx context.Context, roleID string) ([]*entity.Permission, error) {
	var perms []*entity.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Order("permissions.code ASC").
		Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *roleGormRepository) AssignPermission(ctx context.Context, roleID, permissionID, grantedBy string) error {
	rp := entity.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	if grantedBy != "" {
		rp.GrantedBy = &grantedBy
	}
	// Use save to handle upsert on composite PK conflict.
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		FirstOrCreate(&rp).Error
}

func (r *roleGormRepository) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&entity.RolePermission{}).Error
}
