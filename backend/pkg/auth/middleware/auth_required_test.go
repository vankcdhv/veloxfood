package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"

	"github.com/gin-gonic/gin"
)

// mockJWTService implements authjwt.JWTService for testing.
type mockJWTService struct {
	claims *authjwt.Claims
	err    error
}

func (m *mockJWTService) Issue(_ context.Context, _ string) (authjwt.TokenPair, error) {
	return authjwt.TokenPair{}, nil
}
func (m *mockJWTService) Parse(_ context.Context, _ string) (*authjwt.Claims, error) {
	return m.claims, m.err
}

// mockAuthStore implements store.AuthStore for testing.
type mockAuthStore struct {
	exists bool
	err    error
}

func (m *mockAuthStore) Whitelist(_ context.Context, _, _ string, _ time.Duration) error { return nil }
func (m *mockAuthStore) Exists(_ context.Context, _ string) (bool, error) {
	return m.exists, m.err
}
func (m *mockAuthStore) Revoke(_ context.Context, _, _ string) error { return nil }
func (m *mockAuthStore) RevokeAll(_ context.Context, _ string) error { return nil }
func (m *mockAuthStore) RotatePair(_ context.Context, _, _, _, _, _ string, _, _ time.Duration) error {
	return nil
}

func newTestEngine(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": authmw.UserIDFromContext(c.Request.Context())})
	})
	return r
}

func TestAuthRequired_MissingHeader(t *testing.T) {
	svc := &mockJWTService{}
	st := &mockAuthStore{}
	r := newTestEngine(authmw.AuthRequired(svc, st))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	svc := &mockJWTService{err: authjwt.ErrTokenInvalid}
	st := &mockAuthStore{}
	r := newTestEngine(authmw.AuthRequired(svc, st))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_RevokedJTI(t *testing.T) {
	claims := &authjwt.Claims{}
	claims.ID = "test-jti"
	claims.UserID = "user-123"

	svc := &mockJWTService{claims: claims}
	st := &mockAuthStore{exists: false} // jti not in whitelist
	r := newTestEngine(authmw.AuthRequired(svc, st))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_StoreError(t *testing.T) {
	claims := &authjwt.Claims{}
	claims.ID = "test-jti"
	claims.UserID = "user-123"

	svc := &mockJWTService{claims: claims}
	st := &mockAuthStore{err: errors.New("redis down")}
	r := newTestEngine(authmw.AuthRequired(svc, st))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	claims := &authjwt.Claims{}
	claims.ID = "test-jti"
	claims.UserID = "user-123"

	svc := &mockJWTService{claims: claims}
	st := &mockAuthStore{exists: true}
	r := newTestEngine(authmw.AuthRequired(svc, st))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
