package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/cache"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
)

const (
	rbacCacheTTL   = 5 * time.Minute
	rbacGlobalKey  = "rbac:user:%s:perms"
	rbacVendorKey  = "rbac:user:%s:vendor:%s:perms"
	rbacVendorGlob = "rbac:user:%s:vendor:*"
)

// RBACUsecase handles permission checks and role assignment with cache-aside.
type RBACUsecase interface {
	HasPermission(ctx context.Context, userID, code string) (bool, error)
	HasVendorPermission(ctx context.Context, userID, vendorID, code string) (bool, error)
	AssignRoleToUser(ctx context.Context, adminID, userID, roleID string, scopeType entity.ScopeType, scopeID *string, expiresAt *time.Time) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID string, scopeType entity.ScopeType, scopeID *string) error
	ListUserRoles(ctx context.Context, userID string) ([]*entity.UserRole, error)
	InvalidateCacheForUser(ctx context.Context, userID string)
	InvalidateCacheForRole(ctx context.Context, roleID string)
}

type rbacUsecase struct {
	rbacRepo repository.RBACRepository
	cache    cache.Cache
}

// NewRBACUsecase constructs RBACUsecase.
func NewRBACUsecase(rbacRepo repository.RBACRepository, c cache.Cache) RBACUsecase {
	return &rbacUsecase{rbacRepo: rbacRepo, cache: c}
}

// HasPermission checks if userID has the given permission code at global scope.
func (u *rbacUsecase) HasPermission(ctx context.Context, userID, code string) (bool, error) {
	key := fmt.Sprintf(rbacGlobalKey, userID)
	codes, err := u.loadPermCodes(ctx, key, userID, entity.ScopeTypeGlobal, nil)
	if err != nil {
		return false, err
	}
	return containsCode(codes, code), nil
}

// HasVendorPermission checks if userID has the given permission code for a specific vendor scope.
func (u *rbacUsecase) HasVendorPermission(ctx context.Context, userID, vendorID, code string) (bool, error) {
	key := fmt.Sprintf(rbacVendorKey, userID, vendorID)
	codes, err := u.loadPermCodes(ctx, key, userID, entity.ScopeTypeVendor, &vendorID)
	if err != nil {
		return false, err
	}
	return containsCode(codes, code), nil
}

// loadPermCodes loads permissions from cache; on miss queries DB and populates cache.
func (u *rbacUsecase) loadPermCodes(ctx context.Context, key, userID string, scopeType entity.ScopeType, scopeID *string) ([]string, error) {
	// Cache HIT
	if raw, err := u.cache.Get(ctx, key); err == nil {
		var codes []string
		if json.Unmarshal(raw, &codes) == nil {
			slog.DebugContext(ctx, "rbac cache hit", "key", key)
			return codes, nil
		}
	}

	// Cache MISS — query DB
	codes, err := u.rbacRepo.ListUserPermissions(ctx, userID, scopeType, scopeID)
	if err != nil {
		return nil, fmt.Errorf("list user permissions: %w", err)
	}

	// Populate cache — ignore set errors (degrade gracefully)
	if raw, merr := json.Marshal(codes); merr == nil {
		if serr := u.cache.Set(ctx, key, raw, rbacCacheTTL); serr != nil {
			slog.WarnContext(ctx, "rbac cache set failed", "key", key, "err", serr)
		}
	}
	return codes, nil
}

func (u *rbacUsecase) AssignRoleToUser(
	ctx context.Context,
	adminID, userID, roleID string,
	scopeType entity.ScopeType,
	scopeID *string,
	expiresAt *time.Time,
) error {
	ur := &entity.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		ExpiresAt: expiresAt,
	}
	if adminID != "" {
		ur.GrantedBy = &adminID
	}
	if err := u.rbacRepo.AssignRoleToUser(ctx, ur); err != nil {
		return fmt.Errorf("assign role to user: %w", err)
	}
	u.InvalidateCacheForUser(ctx, userID)
	return nil
}

func (u *rbacUsecase) RemoveRoleFromUser(
	ctx context.Context,
	userID, roleID string,
	scopeType entity.ScopeType,
	scopeID *string,
) error {
	if err := u.rbacRepo.RemoveRoleFromUser(ctx, userID, roleID, scopeType, scopeID); err != nil {
		return fmt.Errorf("remove role from user: %w", err)
	}
	u.InvalidateCacheForUser(ctx, userID)
	return nil
}

func (u *rbacUsecase) ListUserRoles(ctx context.Context, userID string) ([]*entity.UserRole, error) {
	return u.rbacRepo.ListUserRoles(ctx, userID)
}

// InvalidateCacheForUser deletes global + all vendor cache keys for a user.
func (u *rbacUsecase) InvalidateCacheForUser(ctx context.Context, userID string) {
	globalKey := fmt.Sprintf(rbacGlobalKey, userID)
	if err := u.cache.Delete(ctx, globalKey); err != nil {
		slog.WarnContext(ctx, "rbac invalidate global cache failed", "user_id", userID, "err", err)
	}

	pattern := fmt.Sprintf(rbacVendorGlob, userID)
	keys, err := u.cache.Scan(ctx, pattern)
	if err != nil {
		slog.WarnContext(ctx, "rbac cache scan failed", "pattern", pattern, "err", err)
		return
	}
	for _, k := range keys {
		if err := u.cache.Delete(ctx, k); err != nil {
			slog.WarnContext(ctx, "rbac invalidate vendor cache failed", "key", k, "err", err)
		}
	}
}

// InvalidateCacheForRole deletes cache for all users assigned to the role.
func (u *rbacUsecase) InvalidateCacheForRole(ctx context.Context, roleID string) {
	userIDs, err := u.rbacRepo.ListUsersByRoleID(ctx, roleID)
	if err != nil {
		slog.WarnContext(ctx, "rbac list users by role failed", "role_id", roleID, "err", err)
		return
	}
	for _, uid := range userIDs {
		u.InvalidateCacheForUser(ctx, uid)
	}
}

func containsCode(codes []string, target string) bool {
	for _, c := range codes {
		if c == target {
			return true
		}
	}
	return false
}
