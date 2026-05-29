package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// RBACRepository provides low-level access for role assignment and permission resolution.
type RBACRepository interface {
	// ListUserPermissions returns permission codes for a user in the given scope.
	// scopeID is nil for global scope queries.
	// Filters out expired user_roles (expires_at IS NULL OR expires_at > NOW()).
	ListUserPermissions(ctx context.Context, userID string, scopeType entity.ScopeType, scopeID *string) ([]string, error)

	// AssignRoleToUser inserts a UserRole record.
	AssignRoleToUser(ctx context.Context, userRole *entity.UserRole) error

	// RemoveRoleFromUser deletes matching UserRole row(s).
	RemoveRoleFromUser(ctx context.Context, userID, roleID string, scopeType entity.ScopeType, scopeID *string) error

	// ListUserRoles returns all active (non-expired) role assignments for a user.
	ListUserRoles(ctx context.Context, userID string) ([]*entity.UserRole, error)

	// ListUsersByRoleID returns user IDs of all users assigned to the given role.
	// Used for cache invalidation when a role's permissions change.
	ListUsersByRoleID(ctx context.Context, roleID string) ([]string, error)
}
