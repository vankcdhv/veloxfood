// Package integration covers the location service end-to-end against a running
// Postgres (location_db) + Redis. Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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
	handlerhttp "project/services/location/internal/handler/http"
	v1 "project/services/location/internal/handler/http/v1"
	"project/services/location/internal/infrastructure/persistence"
	"project/services/location/internal/usecase"

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
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/location.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

type testEnv struct {
	engine *gin.Engine
	db     *gorm.DB
	token  string
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
	// Isolated test DB (location_db_test) — never touches the dev location_db.
	db := testutil.SetupTestDB(t, cfg, migrationsPath())
	tables := []string{"customer_locations", "rooms", "floors", "buildings"}
	testutil.Truncate(t, db, tables...)
	t.Cleanup(func() { testutil.Truncate(t, db, tables...) })

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		t.Skipf("redis unavailable (%v)", err)
	}

	// Mint a token for a fake user (auth verifies via the same in-process jwtSvc).
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
	pair, err := jwtSvc.Issue(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	ctx := context.Background()
	_ = authStore.Whitelist(ctx, pair.JTIAccess, "00000000-0000-0000-0000-000000000001", pair.AccessTTL)

	locRepo := persistence.NewLocationGormRepository(db)
	custRepo := persistence.NewCustomerLocationGormRepository(db)
	locUC := usecase.NewLocationUsecase(locRepo)
	custUC := usecase.NewCustomerLocationUsecase(custRepo, locRepo)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		LocationHandler:      v1.NewLocationHandler(locUC),
		AdminLocationHandler: v1.NewAdminLocationHandler(locUC),
		MeLocationHandler:    v1.NewMeLocationHandler(custUC),
		AuthMiddleware:       authmw.AuthRequired(jwtSvc, authStore),
		PermChecker:          allowChecker{},
	})

	return &testEnv{engine: engine, db: db, token: pair.Access}
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

// dataID extracts response.data.id from the standard JSON envelope.
func dataID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Data struct {
			ID string `json:"ID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v — body: %s", err, w.Body.String())
	}
	return resp.Data.ID
}

func TestLocationTree_AdminAndCustomer(t *testing.T) {
	env := setup(t)

	// Admin: create building → floor → room.
	w := env.do(t, "POST", "/api/v1/admin/buildings", `{"name":"Toà A","address":"123 X"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create building: %d — %s", w.Code, w.Body.String())
	}
	buildingID := dataID(t, w)

	w = env.do(t, "POST", "/api/v1/admin/buildings/"+buildingID+"/floors", `{"name":"Tầng 1","sort_order":1}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create floor: %d — %s", w.Code, w.Body.String())
	}
	floorID := dataID(t, w)

	w = env.do(t, "POST", "/api/v1/admin/floors/"+floorID+"/rooms", `{"code":"P101","name":"Phòng 101"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create room: %d — %s", w.Code, w.Body.String())
	}
	roomID := dataID(t, w)

	// Customer: browse + save a default location.
	w = env.do(t, "GET", "/api/v1/locations/buildings", "")
	if w.Code != http.StatusOK {
		t.Fatalf("browse buildings: %d — %s", w.Code, w.Body.String())
	}

	w = env.do(t, "POST", "/api/v1/me/locations",
		`{"room_id":"`+roomID+`","label":"Văn phòng","is_default":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("add location: %d — %s", w.Code, w.Body.String())
	}

	w = env.do(t, "GET", "/api/v1/me/locations", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list locations: %d — %s", w.Code, w.Body.String())
	}

	// Verify in DB: one customer_location, default true.
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM customer_locations WHERE customer_id = ? AND is_default", "00000000-0000-0000-0000-000000000001").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 default location, got %d", count)
	}
}
