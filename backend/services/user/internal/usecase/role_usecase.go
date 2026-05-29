package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// RoleUsecase manages role CRUD and role-permission assignments.
type RoleUsecase interface {
	List(ctx context.Context, filter repository.RoleFilter) ([]*entity.Role, error)
	GetByID(ctx context.Context, id string) (*entity.Role, error)
	Create(ctx context.Context, in CreateRoleInput) (*entity.Role, error)
	Update(ctx context.Context, id string, in UpdateRoleInput) (*entity.Role, error)
	Delete(ctx context.Context, id string) error

	ListPermissions(ctx context.Context, roleID string) ([]*entity.Permission, error)
	AssignPermission(ctx context.Context, roleID, permissionID, grantedBy string) error
	RevokePermission(ctx context.Context, roleID, permissionID string) error
}

// CreateRoleInput is the validated payload for creating a new role.
type CreateRoleInput struct {
	Code        string
	Name        string
	Description *string
	ScopeType   entity.ScopeType
}

// UpdateRoleInput is the validated payload for patching a role.
type UpdateRoleInput struct {
	Name        *string
	Description *string
}

type roleUsecase struct {
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
	rbacUC   RBACUsecase
}

// NewRoleUsecase constructs RoleUsecase with its dependencies.
func NewRoleUsecase(
	roleRepo repository.RoleRepository,
	permRepo repository.PermissionRepository,
	rbacUC RBACUsecase,
) RoleUsecase {
	return &roleUsecase{
		roleRepo: roleRepo,
		permRepo: permRepo,
		rbacUC:   rbacUC,
	}
}

func (u *roleUsecase) List(ctx context.Context, filter repository.RoleFilter) ([]*entity.Role, error) {
	roles, err := u.roleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return roles, nil
}

func (u *roleUsecase) GetByID(ctx context.Context, id string) (*entity.Role, error) {
	role, err := u.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	return role, nil
}

func (u *roleUsecase) Create(ctx context.Context, in CreateRoleInput) (*entity.Role, error) {
	if _, err := u.roleRepo.GetByCode(ctx, in.Code); err == nil {
		return nil, ErrRoleAlreadyExists
	}

	scopeType := in.ScopeType
	if scopeType == "" {
		scopeType = entity.ScopeTypeGlobal
	}

	role := &entity.Role{
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		ScopeType:   scopeType,
		IsSystem:    false,
	}
	if err := u.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	slog.InfoContext(ctx, "role created", "role_id", role.ID, "code", role.Code)
	return role, nil
}

func (u *roleUsecase) Update(ctx context.Context, id string, in UpdateRoleInput) (*entity.Role, error) {
	role, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		role.Name = *in.Name
	}
	if in.Description != nil {
		role.Description = in.Description
	}

	if err := u.roleRepo.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}
	return role, nil
}

func (u *roleUsecase) Delete(ctx context.Context, id string) error {
	role, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrCannotDeleteSystemRole
	}
	// Invalidate cache for all users who had this role before deleting.
	u.rbacUC.InvalidateCacheForRole(ctx, id)

	if err := u.roleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	slog.InfoContext(ctx, "role deleted", "role_id", id)
	return nil
}

func (u *roleUsecase) ListPermissions(ctx context.Context, roleID string) ([]*entity.Permission, error) {
	if _, err := u.GetByID(ctx, roleID); err != nil {
		return nil, err
	}
	return u.roleRepo.ListPermissions(ctx, roleID)
}

func (u *roleUsecase) AssignPermission(ctx context.Context, roleID, permissionID, grantedBy string) error {
	if _, err := u.GetByID(ctx, roleID); err != nil {
		return err
	}
	if _, err := u.permRepo.GetByID(ctx, permissionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionNotFound
		}
		return fmt.Errorf("get permission: %w", err)
	}
	if err := u.roleRepo.AssignPermission(ctx, roleID, permissionID, grantedBy); err != nil {
		return fmt.Errorf("assign permission: %w", err)
	}
	// Invalidate cache for users who have this role — they now have a new permission.
	u.rbacUC.InvalidateCacheForRole(ctx, roleID)
	return nil
}

func (u *roleUsecase) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	if _, err := u.GetByID(ctx, roleID); err != nil {
		return err
	}
	if err := u.roleRepo.RevokePermission(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}
	// Invalidate cache — affected users no longer have this permission.
	u.rbacUC.InvalidateCacheForRole(ctx, roleID)
	return nil
}
