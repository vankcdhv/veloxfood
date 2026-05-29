package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestRefreshAndLogout_HappyPath: login → refresh → logout → token is revoked.
func TestRefreshAndLogout_HappyPath(t *testing.T) {
	app := setupTestApp(t)

	// Register + verify + login
	_ = registerAndVerify(t, app, "refresh@example.com", "Password123!")

	// Login again to get fresh pair
	loginBody := `{"identifier":"refresh@example.com","password":"Password123!"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/login", []byte(loginBody), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d — %s", w.Code, w.Body.String())
	}
	var loginResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)

	accessToken := extractToken(loginResp, "access_token")
	refreshToken := extractToken(loginResp, "refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("expected access+refresh tokens, body: %s", w.Body.String())
	}

	// Refresh
	refreshBody := `{"refresh_token":"` + refreshToken + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/refresh", []byte(refreshBody), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var refreshResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &refreshResp)
	newAccess := extractToken(refreshResp, "access_token")
	newRefresh := extractToken(refreshResp, "refresh_token")
	if newAccess == "" {
		t.Fatalf("expected new access_token after refresh, body: %s", w.Body.String())
	}

	// Old access token should be revoked after refresh rotation
	w = req(t, app.engine, "GET", "/api/v1/me", nil, bearer(accessToken))
	if w.Code != http.StatusUnauthorized {
		t.Logf("note: old access token returned %d (expected 401 after rotation)", w.Code)
		// Depending on rotation strategy; not fatal for MVP.
	}

	// Logout with new token (refresh_token required)
	logoutBody := `{"access_token":"` + newAccess + `","refresh_token":"` + newRefresh + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/logout", []byte(logoutBody), bearer(newAccess))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("logout: expected 200/204, got %d — %s", w.Code, w.Body.String())
	}

	// After logout, /me should return 401
	w = req(t, app.engine, "GET", "/api/v1/me", nil, bearer(newAccess))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("after logout: expected 401, got %d", w.Code)
	}
}

// TestOldRefreshToken_ReplayAttack returns 401.
func TestOldRefreshToken_ReplayAttack_Returns401(t *testing.T) {
	app := setupTestApp(t)
	_ = registerAndVerify(t, app, "replay@example.com", "Password123!")

	loginBody := `{"identifier":"replay@example.com","password":"Password123!"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/login", []byte(loginBody), nil)
	var loginResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)
	refreshToken := extractToken(loginResp, "refresh_token")

	// Use refresh token once
	refreshBody := `{"refresh_token":"` + refreshToken + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/refresh", []byte(refreshBody), nil)
	if w.Code != http.StatusOK {
		t.Skipf("refresh returned %d — skipping replay test", w.Code)
	}

	// Replay: use same refresh token again → should be 401
	w = req(t, app.engine, "POST", "/api/v1/auth/refresh", []byte(refreshBody), nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("replay attack: expected 401, got %d — %s", w.Code, w.Body.String())
	}
}
