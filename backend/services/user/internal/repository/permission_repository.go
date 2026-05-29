package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// PermissionFilter controls list queries for permissions.
type PermissionFilter struct {
	Resource *string
	Action   *string
	IsSystem *bool
	Limit    int
	Offset   int
}

// PermissionRepository is the data-access contract for Permission aggregates.
type PermissionRepository interface {
	List(ctx context.Context, filter PermissionFilter) ([]*entity.Permission, error)
	GetByID(ctx context.Context, id string) (*entity.Permission, error)
	GetByCode(ctx context.Context, code string) (*entity.Permission, error)
	Create(ctx context.Context, perm *entity.Permission) error
	Update(ctx context.Context, perm *entity.Permission) error
	// Delete removes the permission row. Caller must verify is_system=false before calling.
	Delete(ctx context.Context, id string) error
}
