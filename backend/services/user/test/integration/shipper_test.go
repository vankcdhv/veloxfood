package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
)

// registerAndLogin runs register → verify (OTP) → login and returns the access
// token + the new user's ID. Fails the test on any non-2xx step.
func registerAndLogin(t *testing.T, app *testApp, email string) (token, userID string) {
	t.Helper()
	pw := "Password123!"

	body := `{"email":"` + email + `","password":"` + pw + `","full_name":"Test"}`
	w := req(t, app.engine, "POST", "/api/v1/auth/register", []byte(body), nil)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("register %s: %d — %s", email, w.Code, w.Body.String())
	}
	otp := app.mc.getOTP()
	if otp == "" {
		t.Fatalf("no OTP captured for %s", email)
	}
	vBody := `{"destination":"` + email + `","code":"` + otp + `"}`
	w = req(t, app.engine, "POST", "/api/v1/auth/verify-register", []byte(vBody), nil)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("verify %s: %d — %s", email, w.Code, w.Body.String())
	}
	w = req(t, app.engine, "POST", "/api/v1/auth/login",
		[]byte(`{"identifier":"`+email+`","password":"`+pw+`"}`), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login %s: %d — %s", email, w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	token = extractToken(resp, "access_token")
	if token == "" {
		t.Fatalf("no access_token for %s: %s", email, w.Body.String())
	}
	if err := app.db.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&userID).Error; err != nil || userID == "" {
		t.Fatalf("lookup user id for %s: %v", email, err)
	}
	return token, userID
}

// multipartPhotos builds a multipart body with id_document + portrait files.
// Each part declares an image content-type — the handler whitelists
// JPEG/PNG/WebP and rejects untyped parts.
func multipartPhotos(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, field := range []string{"id_document", "portrait"} {
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, field+".jpg"))
		hdr.Set("Content-Type", "image/jpeg")
		fw, err := mw.CreatePart(hdr)
		if err != nil {
			t.Fatalf("create form file %s: %v", field, err)
		}
		_, _ = fw.Write([]byte("fake-image-bytes-" + field))
	}
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

// TestShipperRegisterAndApprove: customer applies as shipper → admin approves →
// shipper profile becomes approved and the user gains the SHIPPER role.
func TestShipperRegisterAndApprove(t *testing.T) {
	app := setupTestApp(t)

	// 1. A customer applies to become a shipper.
	custToken, custID := registerAndLogin(t, app, "applicant@example.com")

	body, contentType := multipartPhotos(t)
	w := req(t, app.engine, "POST", "/api/v1/shipper/register", body.Bytes(), map[string]string{
		"Authorization": "Bearer " + custToken,
		"Content-Type":  contentType,
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("shipper register: %d — %s", w.Code, w.Body.String())
	}

	// 2. DB: profile exists, status pending.
	var status string
	if err := app.db.Raw("SELECT status FROM shipper_profiles WHERE user_id = ?", custID).Scan(&status).Error; err != nil {
		t.Fatalf("query shipper profile: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected pending, got %q", status)
	}

	// 3. Create an admin and grant the ADMIN role directly.
	adminToken, adminID := registerAndLogin(t, app, "ops.admin@example.com")
	if err := app.db.Exec(
		`INSERT INTO user_roles (user_id, role_id, scope_type, granted_by)
		 SELECT ?, id, 'global', ? FROM roles WHERE code = 'ADMIN'`,
		adminID, adminID,
	).Error; err != nil {
		t.Fatalf("grant ADMIN role: %v", err)
	}

	// 4. Admin approves the shipper.
	w = req(t, app.engine, "POST", "/api/v1/admin/shippers/"+custID+"/approve", nil, bearer(adminToken))
	if w.Code != http.StatusOK {
		t.Fatalf("approve shipper: %d — %s", w.Code, w.Body.String())
	}

	// 5. DB: profile approved + user has SHIPPER role.
	if err := app.db.Raw("SELECT status FROM shipper_profiles WHERE user_id = ?", custID).Scan(&status).Error; err != nil {
		t.Fatalf("re-query shipper profile: %v", err)
	}
	if status != "approved" {
		t.Fatalf("expected approved, got %q", status)
	}

	var roleCount int64
	app.db.Raw(
		`SELECT COUNT(*) FROM user_roles ur JOIN roles r ON ur.role_id = r.id
		 WHERE ur.user_id = ? AND r.code = 'SHIPPER'`, custID,
	).Scan(&roleCount)
	if roleCount != 1 {
		t.Fatalf("expected SHIPPER role granted, got count=%d", roleCount)
	}
}
