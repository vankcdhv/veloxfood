package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestRegisterLogin_HappyPath: register → capture OTP → verify → login → GET /me.
func TestRegisterLogin_HappyPath(t *testing.T) {
	app := setupTestApp(t)

	// 1. Register
	regBody := `{"email":"test@example.com","password":"Password123!","full_name":"Test User"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/register", []byte(regBody), nil)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("register: expected 200/201, got %d — body: %s", w.Code, w.Body.String())
	}

	// 2. Capture OTP from mock mailer
	otpCode := app.mc.getOTP()
	if otpCode == "" {
		t.Fatal("expected OTP to be captured by mock mailer")
	}

	// 3. Verify registration
	verifyBody := `{"destination":"test@example.com","code":"` + otpCode + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/verify-register", []byte(verifyBody), nil)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("verify-register: expected 200/201, got %d — body: %s", w.Code, w.Body.String())
	}

	var verifyResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("parse verify response: %v", err)
	}

	// 4. Login
	loginBody := `{"identifier":"test@example.com","password":"Password123!"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/login", []byte(loginBody), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var loginResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("parse login response: %v", err)
	}

	accessToken := extractToken(loginResp, "access_token")
	if accessToken == "" {
		t.Fatalf("expected access_token in login response, got: %s", w.Body.String())
	}

	// 4b. Login must also issue httpOnly auth cookies.
	cookies := w.Result().Cookies()
	var accessCookie string
	gotAccess, gotRefresh := false, false
	for _, ck := range cookies {
		switch ck.Name {
		case "access_token":
			gotAccess, accessCookie = true, ck.Value
			if !ck.HttpOnly {
				t.Error("access_token cookie must be HttpOnly")
			}
		case "refresh_token":
			gotRefresh = true
		}
	}
	if !gotAccess || !gotRefresh {
		t.Fatalf("expected access_token+refresh_token cookies, got access=%v refresh=%v", gotAccess, gotRefresh)
	}

	// 5. GET /me with Bearer token
	w = req(t, app.engine, "GET", "/api/v1/me", nil, bearer(accessToken))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /me: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// 5b. GET /me must also authenticate via the access_token cookie alone.
	w = req(t, app.engine, "GET", "/api/v1/me", nil, map[string]string{"Cookie": "access_token=" + accessCookie})
	if w.Code != http.StatusOK {
		t.Fatalf("GET /me via cookie: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// 6. Verify audit log has auth.register + auth.login_success
	if c := auditRowCount(t, app.db, "auth.register"); c < 1 {
		t.Error("expected auth.register audit log row")
	}
	if c := auditRowCount(t, app.db, "auth.login_success"); c < 1 {
		t.Error("expected auth.login_success audit log row")
	}
}

// TestLogin_WrongPassword_Returns401.
func TestLogin_WrongPassword_Returns401(t *testing.T) {
	app := setupTestApp(t)

	// Register + verify first
	registerAndVerify(t, app, "wrong@example.com", "Password123!")

	// Login with wrong password
	loginBody := `{"identifier":"wrong@example.com","password":"WrongPass999!"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/login", []byte(loginBody), nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

// registerAndVerify is a helper to register + OTP verify a user.
func registerAndVerify(t *testing.T, app *testApp, email, password string) string {
	t.Helper()

	regBody := `{"email":"` + email + `","password":"` + password + `","full_name":"Test"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/register", []byte(regBody), nil)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("register: expected 200/201, got %d — body: %s", w.Code, w.Body.String())
	}

	otpCode := app.mc.getOTP()
	if otpCode == "" {
		t.Fatal("OTP not captured")
	}

	verifyBody := `{"destination":"` + email + `","code":"` + otpCode + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/verify-register", []byte(verifyBody), nil)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("verify-register: expected 200/201, got %d — body: %s", w.Code, w.Body.String())
	}

	// Login and return access token
	loginBody := `{"identifier":"` + email + `","password":"` + password + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/login", []byte(loginBody), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return extractToken(resp, "access_token")
}

// extractToken tries to find access_token nested in a response map.
func extractToken(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Try data.access_token (wrapped response)
	if data, ok := m["data"].(map[string]any); ok {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	// Try flattened keys
	for _, v := range m {
		if inner, ok := v.(map[string]any); ok {
			if s := extractToken(inner, key); s != "" {
				return s
			}
		}
	}
	return ""
}

// extractString finds any string value for a key recursively.
func extractString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case string:
			return s
		}
	}
	if data, ok := m["data"].(map[string]any); ok {
		return extractString(data, key)
	}
	return ""
}

// extractTokenFromBody parses a JSON body and returns access_token.
func extractTokenFromBody(t *testing.T, body string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	tok := extractToken(m, "access_token")
	if tok == "" {
		// Try refresh
		tok = extractToken(m, "refresh_token")
	}
	return tok
}

// containsAuditAction checks if body indicates an audit row was written (test only).
func containsAuditAction(body, action string) bool {
	return strings.Contains(body, action)
}
