package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"
)

// ---- mock UserRepository (shared across usecase tests in this package) ----

type mockUserRepo struct {
	user *entity.User
	err  error
}

func (m *mockUserRepo) Create(_ context.Context, _ *entity.User) error { return nil }
func (m *mockUserRepo) GetByID(_ context.Context, _ string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) { return nil, nil }
func (m *mockUserRepo) Update(_ context.Context, u *entity.User) error               { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ string) error                     { return nil }
func (m *mockUserRepo) List(_ context.Context, _, _ int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) FindByEmailOrPhone(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateStatus(_ context.Context, _ string, _ entity.UserStatus) error {
	return nil
}
func (m *mockUserRepo) IncrementFailedAttempts(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) ResetFailedAttempts(_ context.Context, _ string) error         { return nil }
func (m *mockUserRepo) SetLockedUntil(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockUserRepo) UpdatePassword(_ context.Context, _, _ string) error           { return nil }
func (m *mockUserRepo) GetByIDs(_ context.Context, _ []string) ([]*entity.User, error) {
	return nil, nil
}

// ---- mock RBACUsecase (minimal — only ListUserRoles needed) ----

type mockRBACUsecase struct {
	roles []*entity.UserRole
	err   error
}

func (m *mockRBACUsecase) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRBACUsecase) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRBACUsecase) AssignRoleToUser(_ context.Context, _, _, _ string, _ entity.ScopeType, _ *string, _ *time.Time) error {
	return nil
}
func (m *mockRBACUsecase) RemoveRoleFromUser(_ context.Context, _, _ string, _ entity.ScopeType, _ *string) error {
	return nil
}
func (m *mockRBACUsecase) ListUserRoles(_ context.Context, _ string) ([]*entity.UserRole, error) {
	return m.roles, m.err
}
func (m *mockRBACUsecase) InvalidateCacheForUser(_ context.Context, _ string) {}
func (m *mockRBACUsecase) InvalidateCacheForRole(_ context.Context, _ string) {}

// ---- helpers ----

func newTestUser() *entity.User {
	email := "test@example.com"
	return &entity.User{
		ID:       "user-uuid-1",
		Email:    &email,
		FullName: "Test User",
		Status:   entity.UserStatusActive,
	}
}

// ---- tests ----

func TestGetMe_Basic(t *testing.T) {
	uc := usecase.NewProfileUsecase(
		&mockUserRepo{user: newTestUser()},
		nil, // membershipRepo optional
		&mockRBACUsecase{roles: []*entity.UserRole{}},
	)

	bundle, err := uc.GetMe(context.Background(), "user-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bundle.User == nil {
		t.Fatal("expected user in bundle")
	}
	if bundle.VendorMemberships == nil {
		t.Error("expected non-nil vendor memberships slice")
	}
}

func TestGetMe_UserNotFound(t *testing.T) {
	uc := usecase.NewProfileUsecase(
		&mockUserRepo{err: errors.New("not found")},
		nil,
		&mockRBACUsecase{},
	)

	_, err := uc.GetMe(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected error when user not found")
	}
}
