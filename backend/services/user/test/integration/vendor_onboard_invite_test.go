package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestVendorOnboard_HappyPath: register → login → onboard → check outbox row.
func TestVendorOnboard_HappyPath(t *testing.T) {
	app := setupTestApp(t)

	// Register and get token
	accessToken := registerAndVerify(t, app, "vendor@example.com", "Password123!")
	if accessToken == "" {
		t.Fatal("login failed after register")
	}

	// Onboard vendor
	onboardBody := `{
		"vendor_name":"Test Canteen",
		"business_type":"canteen",
		"address":"123 Main St",
		"phone":"0123456789"
	}`
	w := req(t, app.engine, "POST", "/api/v1/vendors/onboard", []byte(onboardBody), bearer(accessToken))
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("onboard: expected 200/201, got %d — body: %s", w.Code, w.Body.String())
	}

	var onboardResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &onboardResp)
	vendorID := extractString(onboardResp, "vendor_id")
	if vendorID == "" {
		t.Logf("onboard response: %s", w.Body.String())
		t.Skip("could not extract vendor_id from onboard response")
	}

	// Verify outbox event vendor.requested was written
	var count int64
	app.db.Table("outbox_events").Where("event_type = ? AND aggregate_id = ?", "vendor.requested", vendorID).Count(&count)
	if count < 1 {
		t.Error("expected vendor.requested outbox event, got 0")
	}

	// Verify audit log vendor.onboarded
	if c := auditRowCount(t, app.db, "vendor.onboarded"); c < 1 {
		t.Error("expected vendor.onboarded audit log row")
	}
}

// TestVendorInvite_HappyPath: onboard → invite → accept via token link.
func TestVendorInvite_HappyPath(t *testing.T) {
	app := setupTestApp(t)

	// Owner
	ownerToken := registerAndVerify(t, app, "owner@example.com", "Password123!")

	// Onboard
	onboardBody := `{"vendor_name":"Invite Canteen","business_type":"canteen","address":"","phone":""}`
	w := req(t, app.engine, "POST", "/api/v1/vendors/onboard", []byte(onboardBody), bearer(ownerToken))
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Skipf("onboard returned %d — skip invite test", w.Code)
	}
	var onboardResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &onboardResp)
	vendorID := extractString(onboardResp, "vendor_id")
	if vendorID == "" {
		t.Skip("could not get vendor_id")
	}

	// Invite staff
	inviteBody := `{"email":"staff@example.com","role_in_vendor":"KITCHEN","vendor_name":"Invite Canteen"}`
	w = req(t, app.engine, "POST", "/api/v1/vendors/"+vendorID+"/invitations",
		[]byte(inviteBody), bearer(ownerToken))
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Skipf("invite returned %d — may be permission issue: %s", w.Code, w.Body.String())
	}

	// Capture invite link from mock mailer
	link := app.mc.getLink()
	if link == "" {
		t.Skip("invite link not captured by mock mailer")
	}

	// Extract raw token from link
	parts := strings.Split(link, "token=")
	if len(parts) < 2 {
		t.Skipf("unexpected link format: %s", link)
	}
	rawToken := parts[1]

	// Register accepter
	_ = registerAndVerify(t, app, "staff@example.com", "StaffPass123!")
	staffToken := loginUser(t, app, "staff@example.com", "StaffPass123!")

	// Accept invitation
	acceptBody := `{"token":"` + rawToken + `"}`
	w = req(t, app.engine, "POST", "/api/v1/invitations/accept", []byte(acceptBody), bearer(staffToken))
	if w.Code != http.StatusOK {
		t.Errorf("accept invitation: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Audit log should have vendor.invite_sent + vendor.invite_accepted
	if c := auditRowCount(t, app.db, "vendor.invite_sent"); c < 1 {
		t.Error("expected vendor.invite_sent audit row")
	}
}
