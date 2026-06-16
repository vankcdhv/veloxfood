// Package integration covers the store service end-to-end against a running
// Postgres (store_db) + Redis. Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/store"
	"project/pkg/config"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/testutil"
	handlerhttp "project/services/store/internal/handler/http"
	v1 "project/services/store/internal/handler/http/v1"
	"project/services/store/internal/infrastructure/persistence"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// allowChecker implements middleware.PermissionChecker, always granting access.
type allowChecker struct{}

func (allowChecker) HasPermission(context.Context, string, string) (bool, error) { return true, nil }
func (allowChecker) HasVendorPermission(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/store.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

type testEnv struct {
	engine *gin.Engine
	db     *gorm.DB
	token  string
	userID string
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("SKIP_INTEGRATION=1")
	}

	cfg, err := config.Load(configPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	// Isolated test DB (store_db_test) — never touches the dev store_db.
	db := testutil.SetupTestDB(t, cfg, migrationsPath())
	tables := []string{
		"outbox_events",
		"operating_hours_change_requests",
		"operating_hours",
		"ship_fee_rules",
		"menu_item_options",
		"combo_items",
		"combos",
		"options",
		"option_groups",
		"menu_items",
		"categories",
		"stores",
	}
	testutil.Truncate(t, db, tables...)
	t.Cleanup(func() { testutil.Truncate(t, db, tables...) })

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		t.Skipf("redis unavailable (%v)", err)
	}

	// Mint a token for a fake user (auth verifies via the same in-process jwtSvc).
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
	userID := "00000000-0000-0000-0000-000000000001"
	pair, err := jwtSvc.Issue(context.Background(), userID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	ctx := context.Background()
	_ = authStore.Whitelist(ctx, pair.JTIAccess, userID, pair.AccessTTL)

	// Initialize repositories and usecases
	storeRepo := persistence.NewStoreGormRepository(db)
	catalogRepo := persistence.NewCatalogGormRepository(db)
	shippingRepo := persistence.NewShippingGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)

	// Stub RoomResolver that returns a fixed building for testing
	roomResolver := &stubLocationResolver{}

	storeUC := usecase.NewStoreUsecase(db, storeRepo, outboxRepo, nil)
	catalogUC := usecase.NewCatalogUsecase(catalogRepo)
	shipFeeUC := usecase.NewShipFeeUsecase(shippingRepo, roomResolver)
	hoursUC := usecase.NewHoursUsecase(db, shippingRepo, outboxRepo)

	// Stub FileUploader for image uploads
	stubUploader := &stubFileUploader{}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		BrowseHandler:  v1.NewStoreBrowseHandler(storeUC, catalogUC, shipFeeUC, hoursUC),
		VendorHandler:  v1.NewVendorStoreHandler(storeUC, catalogUC, shipFeeUC, hoursUC, allowChecker{}, stubUploader),
		AdminHandler:   v1.NewAdminStoreHandler(storeUC, hoursUC),
		AuthMiddleware: authmw.AuthRequired(jwtSvc, authStore),
		PermChecker:    allowChecker{},
	})

	return &testEnv{engine: engine, db: db, token: pair.Access, userID: userID}
}

func (e *testEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

// dataID extracts response.data.ID from the standard JSON envelope.
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

type stubLocationResolver struct{}

func (s *stubLocationResolver) GetRoom(_ context.Context, _ string) (floorID, buildingID string, found bool, err error) {
	return "floor-default-001", "building-default-001", true, nil
}

func (s *stubLocationResolver) GetFloor(_ context.Context, _ string) (buildingID string, found bool, err error) {
	return "building-default-001", true, nil
}

func (s *stubLocationResolver) GetBuilding(_ context.Context, _ string) (found bool, err error) {
	return true, nil
}

type stubFileUploader struct{}

func (s *stubFileUploader) Put(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) (string, error) {
	// Stub: return a mock URL
	return "https://storage.example.com/" + objectKey, nil
}

// ── Integration Tests ─────────────────────────────────────────────────────────

func TestAdminCreateStore(t *testing.T) {
	env := setup(t)

	// vendor_id and owner_user_id must be valid UUIDs — stores columns are type uuid.
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0001-0001-0001-000000000001",
		"owner_user_id":"bbbbbbbb-0001-0001-0001-000000000001",
		"name":"Test Store"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create store: %d — %s", w.Code, w.Body.String())
	}
	storeID := dataID(t, w)
	if storeID == "" {
		t.Fatal("expected store ID in response")
	}

	// Verify store was created in DB
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM stores WHERE id = ?", storeID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 store in DB, got %d", count)
	}

	// Verify outbox event was created
	var eventCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = ? AND event_type = ?", storeID, "store.created").Scan(&eventCount)
	if eventCount != 1 {
		t.Fatalf("expected 1 outbox event for store.created, got %d", eventCount)
	}
}

func TestVendorCreateCategory(t *testing.T) {
	env := setup(t)

	// Create a store first
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0002-0002-0002-000000000002",
		"owner_user_id":"bbbbbbbb-0002-0002-0002-000000000002",
		"name":"Vendor Store"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create store: %d", w.Code)
	}
	storeID := dataID(t, w)

	// Create a category
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/categories", `{
		"name":"Soups",
		"sort_order":1
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create category: %d — %s", w.Code, w.Body.String())
	}
	catID := dataID(t, w)

	if catID == "" {
		t.Fatal("expected category ID in response")
	}

	// Verify in DB
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM categories WHERE id = ? AND store_id = ?", catID, storeID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 category in DB, got %d", count)
	}
}

func TestVendorCreateAndToggleMenuItem(t *testing.T) {
	env := setup(t)

	// Create store
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0003-0003-0003-000000000003",
		"owner_user_id":"bbbbbbbb-0003-0003-0003-000000000003",
		"name":"Menu Store"
	}`)
	storeID := dataID(t, w)

	// Create category
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/categories", `{
		"name":"Dishes",
		"sort_order":1
	}`)
	catID := dataID(t, w)

	// Create menu item
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/menu", `{
		"category_id":"`+catID+`",
		"name":"Pho Beef",
		"description":"Vietnamese soup",
		"price":50000,
		"tags":"bestseller,new"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create menu item: %d — %s", w.Code, w.Body.String())
	}
	itemID := dataID(t, w)

	// Verify default status is "on"
	var status string
	env.db.Raw("SELECT status FROM menu_items WHERE id = ?", itemID).Scan(&status)
	if status != "on" {
		t.Errorf("expected status 'on', got %s", status)
	}

	// Toggle to off
	w = env.do(t, "PATCH", "/api/v1/stores/"+storeID+"/menu/"+itemID+"/status", `{
		"status":"off"
	}`)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle menu item: %d — %s", w.Code, w.Body.String())
	}

	// Verify status changed
	env.db.Raw("SELECT status FROM menu_items WHERE id = ?", itemID).Scan(&status)
	if status != "off" {
		t.Errorf("expected status 'off', got %s", status)
	}
}

func TestPublicBrowseStores(t *testing.T) {
	env := setup(t)

	// Create two stores with distinct valid UUIDs.
	browseVendors := []string{
		"aaaaaaaa-0004-0004-0004-000000000001",
		"aaaaaaaa-0004-0004-0004-000000000002",
	}
	browseOwners := []string{
		"bbbbbbbb-0004-0004-0004-000000000001",
		"bbbbbbbb-0004-0004-0004-000000000002",
	}
	for i := 0; i < 2; i++ {
		w := env.do(t, "POST", "/api/v1/admin/stores", `{
			"vendor_id":"`+browseVendors[i]+`",
			"owner_user_id":"`+browseOwners[i]+`",
			"name":"Browse Store `+string(rune('1'+i))+`"
		}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("create store %d: %d", i+1, w.Code)
		}
	}

	// Browse stores
	w := env.do(t, "GET", "/api/v1/stores", "")
	if w.Code != http.StatusOK {
		t.Fatalf("browse stores: %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID   string `json:"ID"`
				Name string `json:"Name"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp.Data.Items) < 2 {
		t.Errorf("expected at least 2 stores, got %d", len(resp.Data.Items))
	}
}

func TestShipFeeRuleCreation(t *testing.T) {
	env := setup(t)

	// Create store
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0005-0005-0005-000000000005",
		"owner_user_id":"bbbbbbbb-0005-0005-0005-000000000005",
		"name":"Ship Fee Store"
	}`)
	storeID := dataID(t, w)

	// ref_id points to a location-service building UUID (no cross-service FK).
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/ship-fees", `{
		"scope":"building",
		"ref_id":"cccccccc-0005-0005-0005-000000000005",
		"unit_fee":5000
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create ship fee rule: %d — %s", w.Code, w.Body.String())
	}
	ruleID := dataID(t, w)

	// Verify in DB
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM ship_fee_rules WHERE id = ? AND scope = ? AND unit_fee = ?", ruleID, "building", 5000).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 ship fee rule in DB, got %d", count)
	}
}

func TestUpdateSaleStatus(t *testing.T) {
	env := setup(t)

	// Create store
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0006-0006-0006-000000000006",
		"owner_user_id":"bbbbbbbb-0006-0006-0006-000000000006",
		"name":"Status Store"
	}`)
	storeID := dataID(t, w)

	// Update sale status to CLOSED_TODAY
	w = env.do(t, "PATCH", "/api/v1/stores/"+storeID+"/sale-status", `{
		"sale_status":"CLOSED_TODAY"
	}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update sale status: %d — %s", w.Code, w.Body.String())
	}

	// Verify in DB
	var status string
	env.db.Raw("SELECT sale_status FROM stores WHERE id = ?", storeID).Scan(&status)
	if status != "CLOSED_TODAY" {
		t.Errorf("expected status CLOSED_TODAY, got %s", status)
	}

	// Verify outbox event
	var eventCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = ? AND event_type = ?", storeID, "store.status_changed").Scan(&eventCount)
	if eventCount != 1 {
		t.Fatalf("expected 1 outbox event for store.status_changed, got %d", eventCount)
	}
}

// TestOperatingHoursWorkflow validates the hours change-request → admin approve flow.
// Per the domain spec, operating-hours and cutoff changes require admin approval before
// taking effect; there is no direct CRUD route for operating_hours by design.
func TestOperatingHoursWorkflow(t *testing.T) {
	env := setup(t)

	// Create store
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-0007-0007-0007-000000000007",
		"owner_user_id":"bbbbbbbb-0007-0007-0007-000000000007",
		"name":"Hours Store"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create store: %d — %s", w.Code, w.Body.String())
	}
	storeID := dataID(t, w)

	// Vendor submits an hours-change request using the structured payload format.
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/hours-change", `{
		"payload": {
			"operating_hours": [
				{"weekday": 1, "open_time": "08:00", "close_time": "22:00"},
				{"weekday": 2, "open_time": "09:00", "close_time": "21:00"}
			],
			"note": "new weekly schedule"
		}
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("submit hours-change: %d — %s", w.Code, w.Body.String())
	}
	reqID := dataID(t, w)
	if reqID == "" {
		t.Fatal("expected change-request ID in submit response")
	}

	// Admin approves the pending change request.
	w = env.do(t, "POST", "/api/v1/admin/stores/"+storeID+"/hours-change/"+reqID+"/approve", "")
	if w.Code != http.StatusOK {
		t.Fatalf("approve hours-change: %d — %s", w.Code, w.Body.String())
	}

	// Verify the change-request row reached status='approved'.
	var status string
	env.db.Raw("SELECT status FROM operating_hours_change_requests WHERE id = ?", reqID).Scan(&status)
	if status != "approved" {
		t.Errorf("expected change-request status 'approved', got %q", status)
	}

	// Verify the approval outbox event was published.
	var evtCount int64
	env.db.Raw(
		"SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = ? AND event_type = ?",
		storeID, "operating_hours.approved",
	).Scan(&evtCount)
	if evtCount != 1 {
		t.Errorf("expected 1 operating_hours.approved outbox event, got %d", evtCount)
	}

	// Verify the operating_hours rows were actually applied to the store.
	var ohCount int64
	env.db.Raw("SELECT COUNT(*) FROM operating_hours WHERE store_id = ?", storeID).Scan(&ohCount)
	if ohCount != 2 {
		t.Errorf("expected 2 operating_hours rows after approve, got %d", ohCount)
	}
	var monOpen string
	env.db.Raw("SELECT open_time FROM operating_hours WHERE store_id = ? AND weekday = 1", storeID).Scan(&monOpen)
	if monOpen != "08:00" {
		t.Errorf("expected Monday open_time '08:00', got %q", monOpen)
	}

	// Verify the read endpoint returns the applied hours.
	w = env.do(t, "GET", "/api/v1/stores/"+storeID+"/hours", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get store hours: %d — %s", w.Code, w.Body.String())
	}
	var hoursResp struct {
		Data struct {
			OperatingHours []struct {
				Weekday   int    `json:"Weekday"`
				OpenTime  string `json:"OpenTime"`
				CloseTime string `json:"CloseTime"`
			} `json:"operating_hours"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &hoursResp); err != nil {
		t.Fatalf("parse hours response: %v — body: %s", err, w.Body.String())
	}
	if len(hoursResp.Data.OperatingHours) != 2 {
		t.Errorf("expected 2 operating_hours in GET /hours response, got %d", len(hoursResp.Data.OperatingHours))
	}
}

func TestStoreForOrderOpenNow(t *testing.T) {
	env := setup(t)

	// Create a store.
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"11111111-1111-1111-1111-111111111111",
		"owner_user_id":"22222222-2222-2222-2222-222222222222",
		"name":"OpenNow Store"
	}`)
	storeID := dataID(t, w)
	if storeID == "" {
		t.Fatalf("create store failed: %d — %s", w.Code, w.Body.String())
	}

	// GetStoreForOrder (the gRPC-facing usecase) populates OpenNow / PrepMinutes.
	hoursUC := usecase.NewHoursUsecase(env.db, persistence.NewShippingGormRepository(env.db), persistence.NewOutboxGormRepository(env.db))
	uc := usecase.NewStoreForOrderUsecase(
		persistence.NewStoreGormRepository(env.db),
		persistence.NewCatalogGormRepository(env.db),
		persistence.NewShippingGormRepository(env.db),
		&stubLocationResolver{},
		hoursUC,
	)
	res, err := uc.GetStoreForOrder(context.Background(), storeID, "ROOM", "")
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if !res.Found {
		t.Fatal("expected store found")
	}
	// No operating_hours seeded → OpenNow should be false (store has no hours configured).
	if res.OpenNow {
		t.Error("expected OpenNow=false when no operating_hours configured")
	}
	// PrepMinutes should be the entity default (15).
	if res.PrepMinutes != 15 {
		t.Errorf("expected PrepMinutes=15, got %d", res.PrepMinutes)
	}
}
