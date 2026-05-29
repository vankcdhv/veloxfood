package usecase_test

import (
	"context"
	"testing"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"gorm.io/gorm"
)

// ---- mock RoleRepository ----

type mockRoleRepo struct {
	roles   []*entity.Role
	created []*entity.Role
	deleted []string
	perms   []*entity.Permission
	byCode  map[string]*entity.Role
}

func newMockRoleRepo(roles ...*entity.Role) *mockRoleRepo {
	m := &mockRoleRepo{
		roles:  roles,
		byCode: make(map[string]*entity.Role),
	}
	for _, r := range roles {
		m.byCode[r.Code] = r
	}
	return m
}

func (m *mockRoleRepo) List(_ context.Context, _ repository.RoleFilter) ([]*entity.Role, error) {
	return m.roles, nil
}
func (m *mockRoleRepo) GetByID(_ context.Context, id string) (*entity.Role, error) {
	for _, r := range m.roles {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockRoleRepo) GetByCode(_ context.Context, code string) (*entity.Role, error) {
	if r, ok := m.byCode[code]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockRoleRepo) Create(_ context.Context, r *entity.Role) error {
	r.ID = "new-role-id"
	m.roles = append(m.roles, r)
	m.created = append(m.created, r)
	return nil
}
func (m *mockRoleRepo) Update(_ context.Context, r *entity.Role) error {
	for i, existing := range m.roles {
		if existing.ID == r.ID {
			m.roles[i] = r
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (m *mockRoleRepo) Delete(_ context.Context, id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRoleRepo) ListPermissions(_ context.Context, _ string) ([]*entity.Permission, error) {
	return m.perms, nil
}
func (m *mockRoleRepo) AssignPermission(_ context.Context, _, _, _ string) error { return nil }
func (m *mockRoleRepo) RevokePermission(_ context.Context, _, _ string) error    { return nil }

// ---- mock PermissionRepository ----

type mockPermRepo struct {
	perms []*entity.Permission
}

func (m *mockPermRepo) List(_ context.Context, _ repository.PermissionFilter) ([]*entity.Permission, error) {
	return m.perms, nil
}
func (m *mockPermRepo) GetByID(_ context.Context, id string) (*entity.Permission, error) {
	for _, p := range m.perms {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockPermRepo) GetByCode(_ context.Context, _ string) (*entity.Permission, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockPermRepo) Create(_ context.Context, _ *entity.Permission) error            { return nil }
func (m *mockPermRepo) Update(_ context.Context, _ *entity.Permission) error            { return nil }
func (m *mockPermRepo) Delete(_ context.Context, _ string) error                        { return nil }

// ---- mock RBAC for role tests ----

type mockRoleRBACUsecase struct {
	invalidatedRoles []string
}

func (m *mockRoleRBACUsecase) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRoleRBACUsecase) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRoleRBACUsecase) AssignRoleToUser(_ context.Context, _, _, _ string, _ entity.ScopeType, _ *string, _ *time.Time) error {
	return nil
}
func (m *mockRoleRBACUsecase) RemoveRoleFromUser(_ context.Context, _, _ string, _ entity.ScopeType, _ *string) error {
	return nil
}
func (m *mockRoleRBACUsecase) ListUserRoles(_ context.Context, _ string) ([]*entity.UserRole, error) {
	return nil, nil
}
func (m *mockRoleRBACUsecase) InvalidateCacheForUser(_ context.Context, _ string) {}
func (m *mockRoleRBACUsecase) InvalidateCacheForRole(_ context.Context, roleID string) {
	m.invalidatedRoles = append(m.invalidatedRoles, roleID)
}

func buildRoleUC(roleRepo *mockRoleRepo, rbac usecase.RBACUsecase) usecase.RoleUsecase {
	return usecase.NewRoleUsecase(roleRepo, &mockPermRepo{}, rbac)
}

// TestRoleList_ReturnsAll verifies List passes through all roles.
func TestRoleList_ReturnsAll(t *testing.T) {
	roles := []*entity.Role{
		{ID: "r1", Code: "ADMIN", Name: "Admin"},
		{ID: "r2", Code: "STAFF", Name: "Staff"},
	}
	rbac := &mockAdminRBACUsecase{}
	uc := buildRoleUC(newMockRoleRepo(roles...), rbac)

	result, err := uc.List(context.Background(), repository.RoleFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 roles, got %d", len(result))
	}
}

// TestRoleDelete_SystemRole_Blocked verifies system roles cannot be deleted.
func TestRoleDelete_SystemRole_Blocked(t *testing.T) {
	sysRole := &entity.Role{ID: "sys-1", Code: "SUPER_ADMIN", Name: "Super Admin", IsSystem: true}
	rbac := &mockAdminRBACUsecase{}
	uc := buildRoleUC(newMockRoleRepo(sysRole), rbac)

	err := uc.Delete(context.Background(), "sys-1")
	if err != usecase.ErrCannotDeleteSystemRole {
		t.Errorf("expected ErrCannotDeleteSystemRole, got: %v", err)
	}
}

// TestRoleDelete_NonSystem_Succeeds verifies non-system roles can be deleted.
func TestRoleDelete_NonSystem_Succeeds(t *testing.T) {
	role := &entity.Role{ID: "custom-1", Code: "CUSTOM", Name: "Custom", IsSystem: false}
	rbac := &mockAdminRBACUsecase{}
	roleRepo := newMockRoleRepo(role)
	uc := buildRoleUC(roleRepo, rbac)

	err := uc.Delete(context.Background(), "custom-1")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
	if len(roleRepo.deleted) != 1 || roleRepo.deleted[0] != "custom-1" {
		t.Error("expected role to be marked deleted")
	}
}

// TestRoleCreate_DuplicateCode_Returns409 verifies duplicate code is rejected.
func TestRoleCreate_DuplicateCode_Returns409(t *testing.T) {
	existing := &entity.Role{ID: "r1", Code: "ADMIN", Name: "Admin"}
	rbac := &mockAdminRBACUsecase{}
	uc := buildRoleUC(newMockRoleRepo(existing), rbac)

	_, err := uc.Create(context.Background(), usecase.CreateRoleInput{
		Code: "ADMIN",
		Name: "Admin Duplicate",
	})
	if err != usecase.ErrRoleAlreadyExists {
		t.Errorf("expected ErrRoleAlreadyExists, got: %v", err)
	}
}

// TestRoleGetByID_NotFound returns ErrRoleNotFound.
func TestRoleGetByID_NotFound_ReturnsError(t *testing.T) {
	rbac := &mockAdminRBACUsecase{}
	uc := buildRoleUC(newMockRoleRepo(), rbac)

	_, err := uc.GetByID(context.Background(), "missing-id")
	if err != usecase.ErrRoleNotFound {
		t.Errorf("expected ErrRoleNotFound, got: %v", err)
	}
}
