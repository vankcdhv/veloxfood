// Package integration contains end-to-end tests requiring running infrastructure
// (Postgres on :5433, Redis on :6380). Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/store"
	rediscache "project/pkg/cache/redis"
	"project/pkg/config"
	pkgmailer "project/pkg/mailer"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/testutil"
	handlerhttp "project/services/user/internal/handler/http"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const skipEnv = "SKIP_INTEGRATION"

// testConfigPath resolves backend/config/config.yaml relative to this source
// file, so integration tests run regardless of the working directory or machine.
func testConfigPath() string {
	_, file, _, _ := runtime.Caller(0) // .../services/user/test/integration/helpers.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/config.yaml"))
}

func userMigrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// fakeUploader implements usecase.FileUploader without touching MinIO —
// it drains the reader and echoes the object key.
type fakeUploader struct{}

func (fakeUploader) Put(_ context.Context, objectKey, _ string, r io.Reader, _ int64) (string, error) {
	_, _ = io.Copy(io.Discard, r)
	return objectKey, nil
}

func skipIfNoInfra(t *testing.T) {
	t.Helper()
	if os.Getenv(skipEnv) == "1" {
		t.Skip("SKIP_INTEGRATION=1: infrastructure not available")
	}
}

// capturingMailer captures OTP codes and invite links for test assertions.
type capturingMailer struct {
	mu       sync.Mutex
	lastOTP  string
	lastLink string
}

func (m *capturingMailer) SendOTP(_ context.Context, _, _, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastOTP = code
	return nil
}
func (m *capturingMailer) SendInvite(_ context.Context, _, link, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastLink = link
	return nil
}
func (m *capturingMailer) SendPasswordReset(_ context.Context, _, link string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastLink = link
	return nil
}

var _ pkgmailer.Mailer = (*capturingMailer)(nil)

func (m *capturingMailer) getOTP() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastOTP
}

func (m *capturingMailer) getLink() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastLink
}

// testApp holds the bootstrapped test state.
type testApp struct {
	db        *gorm.DB
	engine    *gin.Engine
	mc        *capturingMailer
	authStore store.AuthStore
}

func setupTestApp(t *testing.T) *testApp {
	t.Helper()
	skipIfNoInfra(t)

	cfg, err := config.Load(testConfigPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// Isolated test DB (user_db_test) — never touches the dev user_db. Seeded
	// roles/permissions come from the migrations run here, so they survive
	// truncateAll (which omits those tables).
	db := testutil.SetupTestDB(t, cfg, userMigrationsPath())

	truncateAll(t, db)
	t.Cleanup(func() { truncateAll(t, db) })

	// Generate RSA keypair on the fly — no secrets/ dependency.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	pubKey := &privKey.PublicKey
	jwtSvc := authjwt.NewRS256Service(privKey, pubKey, 15*time.Minute, 168*time.Hour)

	authRedis, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		t.Skipf("Redis unavailable (%v) — set SKIP_INTEGRATION=1 to suppress", err)
	}

	cache, _ := rediscache.New(cfg.Redis)
	mc := &capturingMailer{}
	auditLogger := audit.NewGormLogger(db)

	// Repos
	userRepo := persistence.NewUserGormRepository(db)
	otpRepo := persistence.NewOTPGormRepository(db)
	resetRepo := persistence.NewPasswordResetGormRepository(db)
	roleRepo := persistence.NewRoleGormRepository(db)
	permRepo := persistence.NewPermissionGormRepository(db)
	rbacRepo := persistence.NewRBACGormRepository(db)
	membershipRepo := persistence.NewVendorMembershipGormRepository(db)
	invitationRepo := persistence.NewInvitationGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)

	otpSvc := usecase.NewOTPService(otpRepo, mc, cfg.OTP.TTL, cfg.OTP.MaxAttempts, cfg.App.Env)
	authUC := usecase.NewAuthUsecase(userRepo, otpRepo, resetRepo, jwtSvc, authRedis, mc, otpSvc, auditLogger)
	rbacUC := usecase.NewRBACUsecase(rbacRepo, cache)
	profileUC := usecase.NewProfileUsecase(userRepo, membershipRepo, rbacUC)
	roleUC := usecase.NewRoleUsecase(roleRepo, permRepo, rbacUC)
	permUC := usecase.NewPermissionUsecase(permRepo)
	onboardUC := usecase.NewVendorOnboardUsecase(db, membershipRepo, roleRepo, outboxRepo, rbacUC, auditLogger)
	staffUC := usecase.NewVendorStaffUsecase(db, invitationRepo, membershipRepo, roleRepo, outboxRepo, rbacUC, mc, "http://localhost:8080", auditLogger)
	adminUserUC := usecase.NewAdminUserUsecase(db, userRepo, outboxRepo, rbacUC, authRedis, auditLogger)
	adminVendorUC := usecase.NewAdminVendorUsecase(db, membershipRepo, outboxRepo, rbacUC, auditLogger)

	// Shipper (register + admin approve) with an in-memory fake uploader.
	shipperRepo := persistence.NewShipperProfileGormRepository(db)
	shipperRegisterUC := usecase.NewShipperRegisterUsecase(db, shipperRepo, outboxRepo, fakeUploader{}, auditLogger)
	adminShipperUC := usecase.NewAdminShipperUsecase(db, shipperRepo, roleRepo, outboxRepo, rbacUC, auditLogger)

	authMW := authmw.AuthRequired(jwtSvc, authRedis)

	routerCfg := handlerhttp.RouterConfig{
		AuthHandler:         v1.NewAuthHandler(authUC, v1.CookieConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 168 * time.Hour}),
		RoleHandler:         v1.NewRoleHandler(roleUC),
		PermissionHandler:   v1.NewPermissionHandler(permUC),
		UserRoleHandler:     v1.NewUserRoleHandler(rbacUC),
		MeHandler:           v1.NewMeHandler(profileUC),
		VendorHandler:       v1.NewVendorHandler(onboardUC, staffUC),
		InvitationHandler:   v1.NewInvitationHandler(staffUC),
		AdminUserHandler:    v1.NewAdminUserHandler(adminUserUC),
		AdminVendorHandler:  v1.NewAdminVendorHandler(adminVendorUC),
		ShipperHandler:      v1.NewShipperHandler(shipperRegisterUC),
		AdminShipperHandler: v1.NewAdminShipperHandler(adminShipperUC),
		AuthMiddleware:      authMW,
		PermChecker:         rbacUC,
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	handlerhttp.RegisterRoutes(engine, routerCfg)

	t.Cleanup(func() { truncateAll(t, db) })

	return &testApp{
		db:        db,
		engine:    engine,
		mc:        mc,
		authStore: authRedis,
	}
}

// truncateAll clears all transient tables. Seeded roles/permissions are preserved
// because they are in tables not listed here (roles, permissions, role_permissions).
func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"audit_logs", "outbox_events",
		"otp_codes", "password_reset_tokens", "invitations",
		"user_roles", "vendor_memberships",
		"shipper_profiles", "identities",
		"users",
	}
	for _, tbl := range tables {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tbl)).Error; err != nil {
			t.Logf("truncate %s: %v", tbl, err)
		}
	}
}

func req(t *testing.T, engine *gin.Engine, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, r)
	return w
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// rsaPrivDER returns PKCS1 DER bytes for the private key.
func rsaPrivDER(key *rsa.PrivateKey) []byte {
	return x509.MarshalPKCS1PrivateKey(key)
}

// rsaPubDER returns PKIX DER bytes for the public key.
func rsaPubDER(key *rsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(key)
}

// auditRowCount returns the number of audit_logs rows for a given action.
func auditRowCount(t *testing.T, db *gorm.DB, action string) int {
	t.Helper()
	var count int64
	db.Table("audit_logs").Where("action = ?", action).Count(&count)
	return int(count)
}
