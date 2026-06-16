// Package integration covers the promotion service end-to-end against a running
// Postgres (promotion_db) + Redis. Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	authstore "project/pkg/auth/store"
	"project/pkg/config"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/testutil"
	grpchandler "project/services/promotion/internal/handler/grpc"
	handlerhttp "project/services/promotion/internal/handler/http"
	v1 "project/services/promotion/internal/handler/http/v1"
	"project/services/promotion/internal/infrastructure/grpcclient"
	"project/services/promotion/internal/infrastructure/persistence"
	"project/services/promotion/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	promotionv1 "project/proto/promotion/v1"
)

// ── Test doubles ─────────────────────────────────────────────────────────────

// allowChecker grants every permission check — avoids real user-service calls.
type allowChecker struct{}

func (allowChecker) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (allowChecker) HasVendorPermission(context.Context, string, string, string) (bool, error) {
	return true, nil
}

// fixedStoreClient returns a fixed vendor/owner pair for any store_id so tests
// can exercise authorisation without a running store service.
type fixedStoreClient struct {
	vendorID    string
	ownerUserID string
}

func (c *fixedStoreClient) GetStoreOwnership(_ context.Context, storeID string) (*grpcclient.StoreOwnership, error) {
	return &grpcclient.StoreOwnership{
		Found:       true,
		VendorID:    c.vendorID,
		OwnerUserID: c.ownerUserID,
		Name:        "Test Store",
	}, nil
}

// ── Test environment ─────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/promotion.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

type testEnv struct {
	engine   *gin.Engine
	db       *gorm.DB
	token    string
	userID   string
	storeID  string
	vendorID string

	// grpcSrv is used for direct in-process RPC calls (no network required).
	grpcSrv *grpchandler.PromotionServiceServer
}

// setupDB provisions the isolated test DB and wires repositories + usecases.
// It does NOT require Redis — suitable for gRPC-only tests.
func setupDB(t *testing.T) (*testEnv, *gorm.DB) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("SKIP_INTEGRATION=1")
	}

	cfg, err := config.Load(configPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db := testutil.SetupTestDB(t, cfg, migrationsPath())
	tables := []string{"promotion_usages", "promotions"}
	testutil.Truncate(t, db, tables...)
	t.Cleanup(func() { testutil.Truncate(t, db, tables...) })

	userID := "00000000-0000-0000-0000-000000000001"
	storeID := "aaaaaaaa-0000-0000-0000-000000000001"
	vendorID := "bbbbbbbb-0000-0000-0000-000000000001"
	storeClient := &fixedStoreClient{vendorID: vendorID, ownerUserID: userID}

	promoRepo := persistence.NewPromotionGormRepository(db)
	usageRepo := persistence.NewPromotionUsageGormRepository(db)
	promoUC := usecase.NewPromotionUsecase(db, promoRepo, usageRepo, storeClient)
	applyUC := usecase.NewApplyUsecase(db, promoRepo, usageRepo)
	grpcSrv := grpchandler.NewPromotionServiceServer(applyUC)

	env := &testEnv{
		db:       db,
		userID:   userID,
		storeID:  storeID,
		vendorID: vendorID,
		grpcSrv:  grpcSrv,
	}

	// Wire a minimal HTTP engine without auth middleware so vendor/public
	// handler tests work without Redis. The allowChecker grants all permissions.
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())

	// Wrap the engine with a fake auth injector so UserIDFromContext works.
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		VendorHandler:  v1.NewVendorPromotionHandler(promoUC, allowChecker{}),
		PublicHandler:  v1.NewPublicPromotionHandler(promoUC),
		// AuthMiddleware nil → RegisterRoutes skips the auth group wrapper, so we
		// must register routes through a no-op auth pass. Use a pass-through below.
		AuthMiddleware: func(c *gin.Context) { c.Next() },
		PermChecker:    allowChecker{},
	})

	env.engine = engine
	env.token = "test-token" // not validated — auth MW is pass-through above

	return env, db
}

// setup is the full setup including Redis-backed JWT auth. Tests requiring real
// token verification use this; pure logic / gRPC tests use setupDB instead.
func setup(t *testing.T) *testEnv {
	t.Helper()
	env, db := setupDB(t)

	cfg, err := config.Load(configPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	authStore, err := authstore.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		// Redis unavailable — fall back to no-auth mode (engine already wired).
		t.Logf("redis unavailable (%v) — using pass-through auth for HTTP tests", err)
		return env
	}

	// Replace engine with one backed by real JWT auth.
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
	pair, err := jwtSvc.Issue(context.Background(), env.userID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	_ = authStore.Whitelist(context.Background(), pair.JTIAccess, env.userID, pair.AccessTTL)

	storeClient := &fixedStoreClient{vendorID: env.vendorID, ownerUserID: env.userID}
	promoRepo := persistence.NewPromotionGormRepository(db)
	usageRepo := persistence.NewPromotionUsageGormRepository(db)
	promoUC := usecase.NewPromotionUsecase(db, promoRepo, usageRepo, storeClient)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		VendorHandler:  v1.NewVendorPromotionHandler(promoUC, allowChecker{}),
		PublicHandler:  v1.NewPublicPromotionHandler(promoUC),
		AuthMiddleware: authmw.AuthRequired(jwtSvc, authStore),
		PermChecker:    allowChecker{},
	})

	env.engine = engine
	env.token = pair.Access
	return env
}

func (e *testEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var b *bytes.Reader
	if body != "" {
		b = bytes.NewReader([]byte(body))
	} else {
		b = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, b)
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

// dataID extracts response.data.ID from the standard JSON envelope (PascalCase).
func dataID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Data struct {
			ID string `json:"ID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v — body: %s", err, w.Body.String())
	}
	return resp.Data.ID
}

// nowStr returns an RFC3339 time offset by d from now.
func nowStr(d time.Duration) string {
	return time.Now().Add(d).UTC().Format(time.RFC3339)
}

// ── Owner CRUD tests ──────────────────────────────────────────────────────────

func TestOwnerCreateAndListPromotions(t *testing.T) {
	env, _ := setupDB(t)

	body := fmt.Sprintf(`{
		"code":"SAVE10",
		"type":"ORDER_DISCOUNT",
		"value_kind":"PERCENT",
		"value":10,
		"min_order":50000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))

	w := env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create promotion: %d — %s", w.Code, w.Body.String())
	}
	promoID := dataID(t, w)
	if promoID == "" {
		t.Fatal("expected promotion ID in response")
	}

	// Verify in DB.
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM promotions WHERE id = ? AND store_id = ?", promoID, env.storeID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 promotion in DB, got %d", count)
	}

	// List
	w = env.do(t, "GET", "/api/v1/stores/"+env.storeID+"/promotions", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list promotions: %d — %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data struct {
			Items []struct{ ID string } `json:"items"`
			Total int64                 `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data.Items) < 1 {
		t.Errorf("expected at least 1 promotion in list, got %d", len(listResp.Data.Items))
	}
}

func TestOwnerUpdatePromotion(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"UPDATE_ME",
		"type":"SHIP_DISCOUNT",
		"value_kind":"AMOUNT",
		"value":5000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))

	w := env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d — %s", w.Code, w.Body.String())
	}
	promoID := dataID(t, w)

	// Patch: update value and status.
	patchBody := fmt.Sprintf(`{
		"value":8000,
		"status":"INACTIVE",
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(48*time.Hour))

	w = env.do(t, "PATCH", "/api/v1/stores/"+env.storeID+"/promotions/"+promoID, patchBody)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d — %s", w.Code, w.Body.String())
	}

	var status string
	var value int64
	env.db.Raw("SELECT status, value FROM promotions WHERE id = ?", promoID).Row().Scan(&status, &value)
	if status != "INACTIVE" {
		t.Errorf("expected INACTIVE, got %s", status)
	}
	if value != 8000 {
		t.Errorf("expected value 8000, got %d", value)
	}
}

func TestOwnerDeletePromotion(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"DELETE_ME",
		"type":"ORDER_DISCOUNT",
		"value_kind":"AMOUNT",
		"value":2000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))

	w := env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)
	promoID := dataID(t, w)

	w = env.do(t, "DELETE", "/api/v1/stores/"+env.storeID+"/promotions/"+promoID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d — %s", w.Code, w.Body.String())
	}

	// deleted_at must be set (soft delete).
	var deletedAt *time.Time
	env.db.Raw("SELECT deleted_at FROM promotions WHERE id = ?", promoID).Scan(&deletedAt)
	if deletedAt == nil {
		t.Error("expected deleted_at to be set after delete")
	}
}

// ── Validate (dry-run) tests ──────────────────────────────────────────────────

func TestValidatePromotion_Valid(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"VALID20",
		"type":"ORDER_DISCOUNT",
		"value_kind":"PERCENT",
		"value":20,
		"min_order":100000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))
	env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)

	validateBody := fmt.Sprintf(`{
		"code":"VALID20",
		"store_id":"%s",
		"subtotal":200000
	}`, env.storeID)

	w := env.do(t, "POST", "/api/v1/promotions/validate", validateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("validate: %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Applicable   bool  `json:"Applicable"`
			ItemDiscount int64 `json:"ItemDiscount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Data.Applicable {
		t.Error("expected Applicable=true")
	}
	// 20% of 200000 = 40000
	if resp.Data.ItemDiscount != 40000 {
		t.Errorf("expected ItemDiscount=40000, got %d", resp.Data.ItemDiscount)
	}
}

func TestValidatePromotion_MinOrderNotMet(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"MINORDER",
		"type":"ORDER_DISCOUNT",
		"value_kind":"AMOUNT",
		"value":10000,
		"min_order":200000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))
	env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)

	validateBody := fmt.Sprintf(`{"code":"MINORDER","store_id":"%s","subtotal":50000}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/promotions/validate", validateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("validate: %d", w.Code)
	}

	var resp struct {
		Data struct {
			Applicable  bool   `json:"Applicable"`
			ErrorReason string `json:"ErrorReason"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Applicable {
		t.Error("expected Applicable=false for min_order not met")
	}
	if resp.Data.ErrorReason == "" {
		t.Error("expected ErrorReason to be non-empty")
	}
}

func TestValidatePromotion_Expired(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"EXPIRED1",
		"type":"ORDER_DISCOUNT",
		"value_kind":"AMOUNT",
		"value":5000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-48*time.Hour), nowStr(-time.Hour))
	env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)

	validateBody := fmt.Sprintf(`{"code":"EXPIRED1","store_id":"%s","subtotal":100000}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/promotions/validate", validateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("validate: %d", w.Code)
	}
	var resp struct {
		Data struct {
			Applicable bool `json:"Applicable"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Applicable {
		t.Error("expected Applicable=false for expired promotion")
	}
}

func TestValidatePromotion_WrongStore(t *testing.T) {
	env, _ := setupDB(t)

	createBody := fmt.Sprintf(`{
		"code":"WRONGST",
		"type":"ORDER_DISCOUNT",
		"value_kind":"AMOUNT",
		"value":5000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, nowStr(-time.Hour), nowStr(24*time.Hour))
	env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)

	// Send validate with a different store_id.
	validateBody := `{"code":"WRONGST","store_id":"cccccccc-9999-9999-9999-000000000001","subtotal":100000}`
	w := env.do(t, "POST", "/api/v1/promotions/validate", validateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("validate: %d", w.Code)
	}
	var resp struct {
		Data struct {
			Applicable bool `json:"Applicable"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Applicable {
		t.Error("expected Applicable=false for wrong store_id")
	}
}

// ── ApplyPromotion / ConfirmUsage / ReleaseUsage tests ───────────────────────

// seedPromotion creates a promotion directly and returns its code.
func seedPromotion(t *testing.T, env *testEnv, code string, promoType string, usageLimit *int) string {
	t.Helper()
	createBody := fmt.Sprintf(`{
		"code":"%s",
		"type":"%s",
		"value_kind":"AMOUNT",
		"value":10000,
		"starts_at":"%s",
		"ends_at":"%s"
	}`, code, promoType, nowStr(-time.Hour), nowStr(24*time.Hour))
	if usageLimit != nil {
		createBody = fmt.Sprintf(`{
			"code":"%s",
			"type":"%s",
			"value_kind":"AMOUNT",
			"value":10000,
			"usage_limit":%d,
			"starts_at":"%s",
			"ends_at":"%s"
		}`, code, promoType, *usageLimit, nowStr(-time.Hour), nowStr(24*time.Hour))
	}
	w := env.do(t, "POST", "/api/v1/stores/"+env.storeID+"/promotions", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed promotion %s: %d — %s", code, w.Code, w.Body.String())
	}
	return code
}

func TestApplyThenConfirmUsage(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "APPLY01", "ORDER_DISCOUNT", nil)

	orderID := "dddddddd-0001-0001-0001-000000000001"
	customerID := "eeeeeeee-0001-0001-0001-000000000001"

	resp, err := env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: customerID,
		Codes:      []string{"APPLY01"},
		Subtotal:   100000,
	})
	if err != nil {
		t.Fatalf("ApplyPromotion: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error_reason: %s", resp.ErrorReason)
	}
	if resp.ItemDiscount != 10000 {
		t.Errorf("expected ItemDiscount=10000, got %d", resp.ItemDiscount)
	}

	// Verify RESERVED row exists.
	var status string
	env.db.Raw("SELECT status FROM promotion_usages WHERE order_id = ?", orderID).Scan(&status)
	if status != "RESERVED" {
		t.Errorf("expected RESERVED, got %s", status)
	}

	// Confirm.
	cResp, err := env.grpcSrv.ConfirmUsage(context.Background(), &promotionv1.ConfirmUsageRequest{OrderId: orderID})
	if err != nil || !cResp.Success {
		t.Fatalf("ConfirmUsage failed: err=%v resp=%v", err, cResp)
	}

	env.db.Raw("SELECT status FROM promotion_usages WHERE order_id = ?", orderID).Scan(&status)
	if status != "CONFIRMED" {
		t.Errorf("expected CONFIRMED, got %s", status)
	}
}

func TestApplyIdempotent(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "IDEMP01", "ORDER_DISCOUNT", nil)

	orderID := "dddddddd-0002-0002-0002-000000000002"

	req := &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: "eeeeeeee-0002-0002-0002-000000000002",
		Codes:      []string{"IDEMP01"},
		Subtotal:   100000,
	}

	r1, err := env.grpcSrv.ApplyPromotion(context.Background(), req)
	if err != nil || !r1.Success {
		t.Fatalf("first apply failed: %v %v", err, r1)
	}

	// Re-apply same order — must return the same breakdown, not a second reservation.
	r2, err := env.grpcSrv.ApplyPromotion(context.Background(), req)
	if err != nil || !r2.Success {
		t.Fatalf("second apply failed: %v %v", err, r2)
	}

	// used_count must still be 1 (not 2).
	var usedCount int
	env.db.Raw("SELECT used_count FROM promotions WHERE code = ? AND store_id = ?", "IDEMP01", env.storeID).Scan(&usedCount)
	if usedCount != 1 {
		t.Errorf("idempotent re-apply: expected used_count=1, got %d", usedCount)
	}
}

func TestReleaseUsageDecrementsUsedCount(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "RELEASE1", "ORDER_DISCOUNT", nil)

	orderID := "dddddddd-0003-0003-0003-000000000003"

	_, err := env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: "eeeeeeee-0003-0003-0003-000000000003",
		Codes:      []string{"RELEASE1"},
		Subtotal:   100000,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	var usedAfterApply int
	env.db.Raw("SELECT used_count FROM promotions WHERE code = ? AND store_id = ?", "RELEASE1", env.storeID).Scan(&usedAfterApply)
	if usedAfterApply != 1 {
		t.Fatalf("expected used_count=1 after apply, got %d", usedAfterApply)
	}

	rResp, err := env.grpcSrv.ReleaseUsage(context.Background(), &promotionv1.ReleaseUsageRequest{OrderId: orderID})
	if err != nil || !rResp.Success {
		t.Fatalf("release failed: %v %v", err, rResp)
	}

	var usedAfterRelease int
	env.db.Raw("SELECT used_count FROM promotions WHERE code = ? AND store_id = ?", "RELEASE1", env.storeID).Scan(&usedAfterRelease)
	if usedAfterRelease != 0 {
		t.Errorf("expected used_count=0 after release, got %d", usedAfterRelease)
	}

	var usageStatus string
	env.db.Raw("SELECT status FROM promotion_usages WHERE order_id = ?", orderID).Scan(&usageStatus)
	if usageStatus != "VOIDED" {
		t.Errorf("expected VOIDED status, got %s", usageStatus)
	}
}

func TestReleaseUsageIdempotent(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "RLIDMP", "ORDER_DISCOUNT", nil)

	orderID := "dddddddd-0004-0004-0004-000000000004"
	_, _ = env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: "eeeeeeee-0004-0004-0004-000000000004",
		Codes:      []string{"RLIDMP"},
		Subtotal:   100000,
	})

	// Release twice — second call must be a no-op (used_count stays 0).
	_, _ = env.grpcSrv.ReleaseUsage(context.Background(), &promotionv1.ReleaseUsageRequest{OrderId: orderID})
	_, _ = env.grpcSrv.ReleaseUsage(context.Background(), &promotionv1.ReleaseUsageRequest{OrderId: orderID})

	var usedCount int
	env.db.Raw("SELECT used_count FROM promotions WHERE code = ? AND store_id = ?", "RLIDMP", env.storeID).Scan(&usedCount)
	if usedCount != 0 {
		t.Errorf("idempotent release: expected used_count=0, got %d", usedCount)
	}
}

// ── Concurrency: usage_limit=1, two racing ApplyPromotion calls ──────────────

func TestConcurrentApply_UsageLimitOne(t *testing.T) {
	env, _ := setupDB(t)
	limit := 1
	seedPromotion(t, env, "RACE001", "ORDER_DISCOUNT", &limit)

	const goroutines = 5
	results := make([]*promotionv1.ApplyPromotionResponse, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			orderID := fmt.Sprintf("ffffffff-0005-0005-0005-%012d", i+1)
			resp, err := env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
				OrderId:    orderID,
				StoreId:    env.storeID,
				CustomerId: fmt.Sprintf("cccccccc-0005-0005-0005-%012d", i+1),
				Codes:      []string{"RACE001"},
				Subtotal:   100000,
			})
			if err != nil {
				results[i] = &promotionv1.ApplyPromotionResponse{Success: false}
				return
			}
			results[i] = resp
		}()
	}
	wg.Wait()

	successCount := 0
	for _, r := range results {
		if r != nil && r.Success {
			successCount++
		}
	}
	if successCount != 1 {
		t.Errorf("usage_limit=1: expected exactly 1 success, got %d", successCount)
	}

	// used_count must be exactly 1.
	var usedCount int
	env.db.Raw("SELECT used_count FROM promotions WHERE code = ? AND store_id = ?", "RACE001", env.storeID).Scan(&usedCount)
	if usedCount != 1 {
		t.Errorf("expected used_count=1 after race, got %d", usedCount)
	}
}

// ── Type stacking enforcement ─────────────────────────────────────────────────

func TestMaxOneOrderDiscountPerApply(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "ORD_A", "ORDER_DISCOUNT", nil)
	seedPromotion(t, env, "ORD_B", "ORDER_DISCOUNT", nil)

	orderID := "dddddddd-0006-0006-0006-000000000006"
	resp, err := env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: "eeeeeeee-0006-0006-0006-000000000006",
		Codes:      []string{"ORD_A", "ORD_B"},
		Subtotal:   100000,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if resp.Success {
		t.Error("expected failure when applying 2 ORDER_DISCOUNT codes to same order")
	}
	if resp.ErrorReason == "" {
		t.Error("expected ErrorReason to be non-empty")
	}
}

func TestOneOrderPlusOneShipAllowed(t *testing.T) {
	env, _ := setupDB(t)
	seedPromotion(t, env, "MIX_ORD", "ORDER_DISCOUNT", nil)
	seedPromotion(t, env, "MIX_SHP", "SHIP_DISCOUNT", nil)

	orderID := "dddddddd-0007-0007-0007-000000000007"
	resp, err := env.grpcSrv.ApplyPromotion(context.Background(), &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    env.storeID,
		CustomerId: "eeeeeeee-0007-0007-0007-000000000007",
		Codes:      []string{"MIX_ORD", "MIX_SHP"},
		Subtotal:   100000,
	})
	if err != nil || !resp.Success {
		t.Fatalf("expected success for 1 ORDER + 1 SHIP: err=%v reason=%s", err, resp.GetErrorReason())
	}
	if resp.ItemDiscount != 10000 {
		t.Errorf("expected ItemDiscount=10000, got %d", resp.ItemDiscount)
	}
	if resp.ShipDiscount != 10000 {
		t.Errorf("expected ShipDiscount=10000, got %d", resp.ShipDiscount)
	}
}
