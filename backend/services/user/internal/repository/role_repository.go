package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// RoleFilter controls list queries for roles.
type RoleFilter struct {
	ScopeType *entity.ScopeType
	IsSystem  *bool
	Limit     int
	Offset    int
}

// RoleRepository is the data-access contract for Role aggregates.
type RoleRepository interface {
	List(ctx context.Context, filter RoleFilter) ([]*entity.Role, error)
	GetByID(ctx context.Context, id string) (*entity.Role, error)
	GetByCode(ctx context.Context, code string) (*entity.Role, error)
	Create(ctx context.Context, role *entity.Role) error
	Update(ctx context.Context, role *entity.Role) error
	// Delete removes the role row. Caller must verify is_system=false before calling.
	Delete(ctx context.Context, id string) error

	// ListPermissions returns permissions attached to the given role.
	ListPermissions(ctx context.Context, roleID string) ([]*entity.Permission, error)
	// AssignPermission adds a permission to a role (upsert by composite PK).
	AssignPermission(ctx context.Context, roleID, permissionID, grantedBy string) error
	// RevokePermission removes a permission from a role.
	RevokePermission(ctx context.Context, roleID, permissionID string) error
}
