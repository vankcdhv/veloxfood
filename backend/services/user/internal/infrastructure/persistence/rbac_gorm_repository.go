package persistence

import (
	"context"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type rbacGormRepository struct {
	db *gorm.DB
}

// NewRBACGormRepository returns a RBACRepository backed by GORM.
func NewRBACGormRepository(db *gorm.DB) repository.RBACRepository {
	return &rbacGormRepository{db: db}
}

// ListUserPermissions JOIN: user_roles → roles → role_permissions → permissions.
// Filters by scope and excludes expired assignments.
func (r *rbacGormRepository) ListUserPermissions(
	ctx context.Context,
	userID string,
	scopeType entity.ScopeType,
	scopeID *string,
) ([]string, error) {
	q := r.db.WithContext(ctx).
		Table("user_roles ur").
		Select("DISTINCT p.code").
		Joins("JOIN roles ro ON ro.id = ur.role_id").
		Joins("JOIN role_permissions rp ON rp.role_id = ro.id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("ur.user_id = ? AND ur.scope_type = ?", userID, scopeType).
		Where("ur.expires_at IS NULL OR ur.expires_at > NOW()")

	if scopeID != nil {
		q = q.Where("ur.scope_id = ?", *scopeID)
	} else {
		q = q.Where("ur.scope_id IS NULL")
	}

	var codes []string
	if err := q.Pluck("p.code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (r *rbacGormRepository) AssignRoleToUser(ctx context.Context, userRole *entity.UserRole) error {
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *rbacGormRepository) RemoveRoleFromUser(
	ctx context.Context,
	userID, roleID string,
	scopeType entity.ScopeType,
	scopeID *string,
) error {
	q := r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ? AND scope_type = ?", userID, roleID, scopeType)

	if scopeID != nil {
		q = q.Where("scope_id = ?", *scopeID)
	} else {
		q = q.Where("scope_id IS NULL")
	}

	return q.Delete(&entity.UserRole{}).Error
}

func (r *rbacGormRepository) ListUserRoles(ctx context.Context, userID string) ([]*entity.UserRole, error) {
	var roles []*entity.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > NOW())", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *rbacGormRepository) ListUsersByRoleID(ctx context.Context, roleID string) ([]string, error) {
	var userIDs []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Where("role_id = ? AND (expires_at IS NULL OR expires_at > NOW())", roleID).
		Pluck("user_id", &userIDs).Error
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}
