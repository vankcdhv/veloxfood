package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authmw "project/pkg/auth/middleware"

	"github.com/gin-gonic/gin"
)

// mockPermissionChecker implements authmw.PermissionChecker for testing.
type mockPermissionChecker struct {
	hasPerm       bool
	hasVendorPerm bool
	err           error
}

func (m *mockPermissionChecker) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return m.hasPerm, m.err
}
func (m *mockPermissionChecker) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return m.hasVendorPerm, m.err
}

func newPermTestEngine(checker authmw.PermissionChecker, code string, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		// Inject user_id into context, simulating AuthRequired
		if userID != "" {
			c.Request = c.Request.WithContext(authmw.WithUserID(c.Request.Context(), userID))
		}
		c.Next()
	}, authmw.PermissionRequired(checker, code), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestPermissionRequired_Allow(t *testing.T) {
	checker := &mockPermissionChecker{hasPerm: true}
	r := newPermTestEngine(checker, "user.read", "user-123")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPermissionRequired_Deny(t *testing.T) {
	checker := &mockPermissionChecker{hasPerm: false}
	r := newPermTestEngine(checker, "user.suspend", "user-123")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestPermissionRequired_NoUserID(t *testing.T) {
	checker := &mockPermissionChecker{hasPerm: true}
	r := newPermTestEngine(checker, "user.read", "") // no user_id

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPermissionRequired_CheckerError(t *testing.T) {
	checker := &mockPermissionChecker{err: errors.New("db error")}
	r := newPermTestEngine(checker, "user.read", "user-123")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func newVendorPermTestEngine(checker authmw.PermissionChecker, code, vendorParam string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/vendors/:vendor_id/test", func(c *gin.Context) {
		c.Request = c.Request.WithContext(authmw.WithUserID(c.Request.Context(), "user-123"))
		c.Next()
	}, authmw.VendorPermissionRequired(checker, code, vendorParam), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestVendorPermissionRequired_InvalidUUID(t *testing.T) {
	checker := &mockPermissionChecker{hasVendorPerm: true}
	r := newVendorPermTestEngine(checker, "vendor_member.invite", "vendor_id")

	req := httptest.NewRequest(http.MethodGet, "/vendors/not-a-uuid/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestVendorPermissionRequired_Allow(t *testing.T) {
	checker := &mockPermissionChecker{hasVendorPerm: true}
	r := newVendorPermTestEngine(checker, "vendor_member.invite", "vendor_id")

	req := httptest.NewRequest(http.MethodGet, "/vendors/550e8400-e29b-41d4-a716-446655440000/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestVendorPermissionRequired_Deny(t *testing.T) {
	checker := &mockPermissionChecker{hasVendorPerm: false}
	r := newVendorPermTestEngine(checker, "vendor_member.invite", "vendor_id")

	req := httptest.NewRequest(http.MethodGet, "/vendors/550e8400-e29b-41d4-a716-446655440000/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
