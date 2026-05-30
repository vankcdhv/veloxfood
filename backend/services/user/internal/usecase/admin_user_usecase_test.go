package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"gorm.io/gorm"
)

// ---- mock AuthStore ----

type mockAuthStore struct {
	revokeAllCalled bool
	revokeAllErr    error
}

func (m *mockAuthStore) Whitelist(_ context.Context, _, _ string, _ time.Duration) error { return nil }
func (m *mockAuthStore) Exists(_ context.Context, _ string) (bool, error)                { return true, nil }
func (m *mockAuthStore) Revoke(_ context.Context, _, _ string) error                     { return nil }
func (m *mockAuthStore) RevokeAll(_ context.Context, _ string) error {
	m.revokeAllCalled = true
	return m.revokeAllErr
}
func (m *mockAuthStore) RotatePair(_ context.Context, _, _, _, _, _ string, _, _ time.Duration) error {
	return nil
}

// ---- mock OutboxRepository ----

type mockAdminOutboxRepo struct {
	events []*entity.OutboxEvent
	err    error
}

func (m *mockAdminOutboxRepo) Append(_ context.Context, _ *gorm.DB, evt *entity.OutboxEvent) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, evt)
	return nil
}

func (m *mockAdminOutboxRepo) Pickup(_ context.Context, _, _ int) ([]*entity.OutboxEvent, error) {
	return nil, nil
}
func (m *mockAdminOutboxRepo) MarkPublished(_ context.Context, _ string) error { return nil }
func (m *mockAdminOutboxRepo) MarkFailed(_ context.Context, _, _ string, _, _ int) error {
	return nil
}

// ---- mock RBACUsecase (extended to track invalidation) ----

type mockAdminRBACUsecase struct {
	invalidatedUsers []string
}

func (m *mockAdminRBACUsecase) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockAdminRBACUsecase) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockAdminRBACUsecase) AssignRoleToUser(_ context.Context, _, _, _ string, _ entity.ScopeType, _ *string, _ *time.Time) error {
	return nil
}
func (m *mockAdminRBACUsecase) RemoveRoleFromUser(_ context.Context, _, _ string, _ entity.ScopeType, _ *string) error {
	return nil
}
func (m *mockAdminRBACUsecase) ListUserRoles(_ context.Context, _ string) ([]*entity.UserRole, error) {
	return nil, nil
}
func (m *mockAdminRBACUsecase) InvalidateCacheForUser(_ context.Context, userID string) {
	m.invalidatedUsers = append(m.invalidatedUsers, userID)
}
func (m *mockAdminRBACUsecase) InvalidateCacheForRole(_ context.Context, _ string) {}

// --- helpers ---

func newAdminUserUCWithDB(t *testing.T, db *gorm.DB, userRepo *mockUserRepo, rbac *mockAdminRBACUsecase, authStore *mockAuthStore) usecase.AdminUserUsecase {
	t.Helper()
	return usecase.NewAdminUserUsecase(
		db,
		userRepo,
		&mockAdminOutboxRepo{},
		rbac,
		authStore,
		audit.NoopLogger{},
	)
}

// TestSuspend_SelfSuspend_ReturnsSelfLockoutError verifies the self-suspend guard.
func TestSuspend_SelfSuspend_ReturnsSelfLockoutError(t *testing.T) {
	rbac := &mockAdminRBACUsecase{}
	authStore := &mockAuthStore{}
	userRepo := &mockUserRepo{
		user: &entity.User{
			ID:     "admin-1",
			Status: entity.UserStatusActive,
		},
	}

	uc := newAdminUserUCWithDB(t, nil, userRepo, rbac, authStore)
	err := uc.Suspend(context.Background(), "admin-1", "admin-1", "test")
	if err == nil {
		t.Fatal("expected ErrCannotActOnSelf, got nil")
	}
	if err != usecase.ErrCannotActOnSelf {
		t.Errorf("expected ErrCannotActOnSelf, got: %v", err)
	}
	// RevokeAll must NOT have been called.
	if authStore.revokeAllCalled {
		t.Error("RevokeAll should not be called when self-suspend guard triggers")
	}
}

// TestSuspend_UserAlreadySuspended returns ErrUserAlreadyInStatus.
func TestSuspend_UserAlreadySuspended_ReturnsConflict(t *testing.T) {
	userRepo := &mockUserRepo{
		user: &entity.User{ID: "user-1", Status: entity.UserStatusSuspended},
	}
	authStore := &mockAuthStore{}
	uc := newAdminUserUCWithDB(t, nil, userRepo, &mockAdminRBACUsecase{}, authStore)

	err := uc.Suspend(context.Background(), "admin-1", "user-1", "again")
	if err != usecase.ErrUserAlreadyInStatus {
		t.Errorf("expected ErrUserAlreadyInStatus, got: %v", err)
	}
}

// TestSuspend_UserNotFound propagates ErrUserNotFound.
func TestSuspend_UserNotFound_PropagatesError(t *testing.T) {
	userRepo := &mockUserRepo{err: fmt.Errorf("record not found")}
	uc := newAdminUserUCWithDB(t, nil, userRepo, &mockAdminRBACUsecase{}, &mockAuthStore{})

	err := uc.Suspend(context.Background(), "admin-1", "missing-user", "reason")
	if err != usecase.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

// TestReactivate_UserAlreadyActive returns ErrUserAlreadyInStatus.
func TestReactivate_UserAlreadyActive_ReturnsConflict(t *testing.T) {
	userRepo := &mockUserRepo{
		user: &entity.User{ID: "user-1", Status: entity.UserStatusActive},
	}
	uc := newAdminUserUCWithDB(t, nil, userRepo, &mockAdminRBACUsecase{}, &mockAuthStore{})

	err := uc.Reactivate(context.Background(), "admin-1", "user-1")
	if err != usecase.ErrUserAlreadyInStatus {
		t.Errorf("expected ErrUserAlreadyInStatus, got: %v", err)
	}
}

// TestReactivate_UserNotFound propagates ErrUserNotFound.
func TestReactivate_UserNotFound_PropagatesError(t *testing.T) {
	userRepo := &mockUserRepo{err: fmt.Errorf("record not found")}
	uc := newAdminUserUCWithDB(t, nil, userRepo, &mockAdminRBACUsecase{}, &mockAuthStore{})

	err := uc.Reactivate(context.Background(), "admin-1", "missing-user")
	if err != usecase.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

// TestRemoveRole_SelfGlobalRemove_ReturnsSelfLockoutError verifies self-role-removal guard.
func TestRemoveRole_SelfGlobalRemove_ReturnsSelfLockoutError(t *testing.T) {
	userRepo := &mockUserRepo{user: &entity.User{ID: "admin-1"}}
	rbac := &mockAdminRBACUsecase{}
	authStore := &mockAuthStore{}
	uc := newAdminUserUCWithDB(t, nil, userRepo, rbac, authStore)

	err := uc.RemoveRole(context.Background(), "admin-1", "admin-1", "role-super", entity.ScopeTypeGlobal, nil)
	if err != usecase.ErrCannotActOnSelf {
		t.Errorf("expected ErrCannotActOnSelf, got: %v", err)
	}
}

// TestRemoveRole_SelfVendorScoped_IsAllowed verifies self-removal of vendor-scoped role is permitted.
func TestRemoveRole_SelfVendorScope_IsAllowed(t *testing.T) {
	userRepo := &mockUserRepo{user: &entity.User{ID: "admin-1"}}
	rbac := &mockAdminRBACUsecase{}
	uc := newAdminUserUCWithDB(t, nil, userRepo, rbac, &mockAuthStore{})

	vendorID := "vendor-1"
	// Vendor-scoped removal: actor == target is allowed (transfer ownership scenario).
	err := uc.RemoveRole(context.Background(), "admin-1", "admin-1", "role-owner", entity.ScopeTypeVendor, &vendorID)
	if err != nil {
		t.Errorf("expected nil for vendor-scoped self-removal, got: %v", err)
	}
}
