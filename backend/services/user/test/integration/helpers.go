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
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/store"
	rediscache "project/pkg/cache/redis"
	"project/pkg/config"
	"project/pkg/database"
	pkgmailer "project/pkg/mailer"
	pkgmiddleware "project/pkg/middleware"
	handlerhttp "project/services/user/internal/handler/http"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const skipEnv = "SKIP_INTEGRATION"

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

	cfg, err := config.Load("/Users/levan/develop/thacsi/distributed_system/project/backend/config/config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		t.Skipf("DB unavailable (%v) — set SKIP_INTEGRATION=1 to suppress", err)
	}

	truncateAll(t, db)

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
	cardRepo := persistence.NewCardGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)
	studentRepo := persistence.NewStudentProfileGormRepository(db)
	facultyRepo := persistence.NewFacultyProfileGormRepository(db)

	otpSvc := usecase.NewOTPService(otpRepo, mc, cfg.OTP.TTL, cfg.OTP.MaxAttempts)
	authUC := usecase.NewAuthUsecase(userRepo, otpRepo, resetRepo, jwtSvc, authRedis, mc, otpSvc, auditLogger)
	rbacUC := usecase.NewRBACUsecase(rbacRepo, cache)
	profileUC := usecase.NewProfileUsecase(userRepo, studentRepo, facultyRepo, membershipRepo, rbacUC)
	cardUC := usecase.NewCardUsecase(cardRepo, userRepo, auditLogger)
	roleUC := usecase.NewRoleUsecase(roleRepo, permRepo, rbacUC)
	permUC := usecase.NewPermissionUsecase(permRepo)
	onboardUC := usecase.NewVendorOnboardUsecase(db, membershipRepo, roleRepo, outboxRepo, rbacUC, auditLogger)
	staffUC := usecase.NewVendorStaffUsecase(db, invitationRepo, membershipRepo, roleRepo, outboxRepo, rbacUC, mc, "http://localhost:8080", auditLogger)
	adminUserUC := usecase.NewAdminUserUsecase(db, userRepo, studentRepo, facultyRepo, outboxRepo, rbacUC, authRedis, auditLogger)
	adminVendorUC := usecase.NewAdminVendorUsecase(db, membershipRepo, outboxRepo, rbacUC, auditLogger)

	authMW := authmw.AuthRequired(jwtSvc, authRedis)

	routerCfg := handlerhttp.RouterConfig{
		AuthHandler:        v1.NewAuthHandler(authUC, v1.CookieConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 168 * time.Hour}),
		RoleHandler:        v1.NewRoleHandler(roleUC),
		PermissionHandler:  v1.NewPermissionHandler(permUC),
		UserRoleHandler:    v1.NewUserRoleHandler(rbacUC),
		MeHandler:          v1.NewMeHandler(profileUC),
		CardAdminHandler:   v1.NewCardAdminHandler(cardUC),
		VendorHandler:      v1.NewVendorHandler(onboardUC, staffUC),
		InvitationHandler:  v1.NewInvitationHandler(staffUC),
		AdminUserHandler:   v1.NewAdminUserHandler(adminUserUC),
		AdminVendorHandler: v1.NewAdminVendorHandler(adminVendorUC),
		AuthMiddleware:     authMW,
		PermChecker:        rbacUC,
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
		"audit_logs", "outbox_events", "card_identifiers",
		"otp_codes", "password_reset_tokens", "invitations",
		"user_roles", "vendor_memberships",
		"student_profiles", "faculty_profiles",
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
