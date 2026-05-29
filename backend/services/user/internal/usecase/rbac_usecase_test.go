package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"
)

// ---- mock RBACRepository ----

type mockRBACRepository struct {
	perms  []string
	err    error
	called bool
}

func (m *mockRBACRepository) ListUserPermissions(_ context.Context, _ string, _ entity.ScopeType, _ *string) ([]string, error) {
	m.called = true
	return m.perms, m.err
}
func (m *mockRBACRepository) AssignRoleToUser(_ context.Context, _ *entity.UserRole) error { return nil }
func (m *mockRBACRepository) RemoveRoleFromUser(_ context.Context, _, _ string, _ entity.ScopeType, _ *string) error {
	return nil
}
func (m *mockRBACRepository) ListUserRoles(_ context.Context, _ string) ([]*entity.UserRole, error) {
	return nil, nil
}
func (m *mockRBACRepository) ListUsersByRoleID(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// ---- mock Cache ----

type mockCache struct {
	store map[string][]byte
	// scanKeys returned by Scan
	scanKeys []string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string][]byte)}
}

func (m *mockCache) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.store[key]
	if !ok {
		return nil, errors.New("cache miss")
	}
	return v, nil
}

func (m *mockCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.store[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *mockCache) Scan(_ context.Context, _ string) ([]string, error) {
	return m.scanKeys, nil
}

// ---- tests ----

func TestHasPermission_CacheMiss_DBHit(t *testing.T) {
	repo := &mockRBACRepository{perms: []string{"user.read", "user.suspend"}}
	cache := newMockCache()
	uc := usecase.NewRBACUsecase(repo, cache)

	ok, err := uc.HasPermission(context.Background(), "user-1", "user.suspend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasPermission=true")
	}
	if !repo.called {
		t.Error("expected DB to be queried on cache miss")
	}
}

func TestHasPermission_CacheHit_NoDB(t *testing.T) {
	repo := &mockRBACRepository{perms: []string{"user.read"}}
	cache := newMockCache()

	// Pre-populate cache with user.suspend
	codes := []string{"user.suspend"}
	raw, _ := json.Marshal(codes)
	_ = cache.Set(context.Background(), "rbac:user:user-1:perms", raw, 5*time.Minute)

	uc := usecase.NewRBACUsecase(repo, cache)
	ok, err := uc.HasPermission(context.Background(), "user-1", "user.suspend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasPermission=true from cache")
	}
	if repo.called {
		t.Error("DB should NOT be queried on cache hit")
	}
}

func TestHasPermission_CacheHit_PermissionMissing(t *testing.T) {
	repo := &mockRBACRepository{}
	cache := newMockCache()

	// Cache has read but not suspend
	codes := []string{"user.read"}
	raw, _ := json.Marshal(codes)
	_ = cache.Set(context.Background(), "rbac:user:user-1:perms", raw, 5*time.Minute)

	uc := usecase.NewRBACUsecase(repo, cache)
	ok, err := uc.HasPermission(context.Background(), "user-1", "user.suspend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected HasPermission=false — permission not in cache")
	}
}

func TestHasPermission_DBError(t *testing.T) {
	repo := &mockRBACRepository{err: errors.New("db down")}
	cache := newMockCache()
	uc := usecase.NewRBACUsecase(repo, cache)

	_, err := uc.HasPermission(context.Background(), "user-1", "user.read")
	if err == nil {
		t.Error("expected error on DB failure")
	}
}

func TestHasVendorPermission_CacheMiss(t *testing.T) {
	repo := &mockRBACRepository{perms: []string{"vendor_member.invite"}}
	cache := newMockCache()
	uc := usecase.NewRBACUsecase(repo, cache)

	ok, err := uc.HasVendorPermission(context.Background(), "user-1", "vendor-abc", "vendor_member.invite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasVendorPermission=true")
	}
	// verify cache was populated
	key := "rbac:user:user-1:vendor:vendor-abc:perms"
	if _, err := cache.Get(context.Background(), key); err != nil {
		t.Error("expected vendor permission to be cached")
	}
}

func TestInvalidateCacheForUser(t *testing.T) {
	repo := &mockRBACRepository{}
	cache := newMockCache()

	// Pre-populate global + vendor keys
	_ = cache.Set(context.Background(), "rbac:user:user-1:perms", []byte(`[]`), 0)
	cache.scanKeys = []string{"rbac:user:user-1:vendor:v1:perms", "rbac:user:user-1:vendor:v2:perms"}
	for _, k := range cache.scanKeys {
		_ = cache.Set(context.Background(), k, []byte(`[]`), 0)
	}

	uc := usecase.NewRBACUsecase(repo, cache)
	uc.InvalidateCacheForUser(context.Background(), "user-1")

	// global key should be gone
	if _, err := cache.Get(context.Background(), "rbac:user:user-1:perms"); err == nil {
		t.Error("expected global cache key to be deleted")
	}
	// vendor keys should be gone
	for _, k := range cache.scanKeys {
		if _, err := cache.Get(context.Background(), k); err == nil {
			t.Errorf("expected vendor cache key %s to be deleted", k)
		}
	}
}
