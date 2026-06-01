// Package integration tests the reporting service end-to-end against a real
// Postgres instance (reporting_db_test). Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	authmw "project/pkg/auth/middleware"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/outbox"
	"project/pkg/testutil"
	reportevent "project/services/reporting/internal/handler/event"
	handlerhttp "project/services/reporting/internal/handler/http"
	v1 "project/services/reporting/internal/handler/http/v1"
	"project/services/reporting/internal/infrastructure/persistence"
	"project/services/reporting/internal/usecase"

	"project/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ── path helpers ──────────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/reporting.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// ── tables cleared between tests ─────────────────────────────────────────────

var allTables = []string{
	"order_facts", "revenue_daily", "user_growth_daily",
	"store_performance", "processed_events",
}

// ── test environment ──────────────────────────────────────────────────────────

type testEnv struct {
	engine        *gin.Engine
	db            *gorm.DB
	orderHandler  *reportevent.OrderEventHandler
	payHandler    *reportevent.PaymentEventHandler
	rvHandler     *reportevent.ReviewVendorEventHandler
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("SKIP_INTEGRATION=1")
	}

	cfg, err := config.Load(configPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db := testutil.SetupTestDB(t, cfg, migrationsPath())
	testutil.Truncate(t, db, allTables...)
	t.Cleanup(func() { testutil.Truncate(t, db, allTables...) })

	projRepo      := persistence.NewProjectionGormRepository(db)
	queryRepo     := persistence.NewQueryGormRepository(db)
	processedRepo := persistence.NewProcessedEventGormRepository(db)
	analyticsUC   := usecase.NewAnalyticsUsecase(queryRepo)

	const fakeUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	// Inject a fixed user ID so auth middleware is bypassed in tests.
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), fakeUserID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		AdminHandler:   v1.NewAdminAnalyticsHandler(analyticsUC),
		StoreHandler:   v1.NewStoreReportHandler(analyticsUC),
		AuthMiddleware: func(c *gin.Context) { c.Next() },
	})

	return &testEnv{
		engine:       engine,
		db:           db,
		orderHandler: reportevent.NewOrderEventHandler(db, projRepo, processedRepo),
		payHandler:   reportevent.NewPaymentEventHandler(db, projRepo, processedRepo),
		rvHandler:    reportevent.NewReviewVendorEventHandler(db, projRepo, processedRepo),
	}
}

// buildMsg wraps data in an outbox.Envelope kafka.Message.
func buildMsg(eventType string, data any) kafka.Message {
	raw, _ := json.Marshal(data)
	env := outbox.Envelope{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Data:       raw,
	}
	b, _ := json.Marshal(env)
	return kafka.Message{Topic: "test", Value: b}
}

func (e *testEnv) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestOrderPlaced_CreatesOrderFact verifies that an order.placed event produces
// an order_fact row and a revenue_daily entry.
func TestOrderPlaced_CreatesOrderFact(t *testing.T) {
	env := setupEnv(t)

	orderID := uuid.NewString()
	storeID := uuid.NewString()
	customerID := uuid.NewString()
	today := time.Now().UTC().Format("2006-01-02")

	msg := buildMsg("order.placed", map[string]any{
		"order_id":       orderID,
		"store_id":       storeID,
		"customer_id":    customerID,
		"fulfillment":    "DELIVERY",
		"payment_method": "COD",
		"items_total":    int64(120000),
		"ship_fee":       int64(20000),
		"discount":       int64(0),
		"grand_total":    int64(140000),
		"items":          []map[string]any{{"date": today, "qty": 1}},
	})

	if err := env.orderHandler.HandleKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("HandleKafkaMessage order.placed: %v", err)
	}

	// order_facts row must exist with PENDING status.
	var status string
	env.db.Raw("SELECT status FROM order_facts WHERE order_id=?", orderID).Scan(&status)
	if status != "PENDING" {
		t.Errorf("expected order_fact status=PENDING, got %q", status)
	}

	// revenue_daily orders_count must be 1.
	var cnt int
	env.db.Raw("SELECT orders_count FROM revenue_daily WHERE store_id=?", storeID).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("expected revenue_daily.orders_count=1, got %d", cnt)
	}
}

// TestOrderPlaced_Idempotent verifies replaying the same event does not
// double-insert the order_fact or increment revenue_daily a second time.
func TestOrderPlaced_Idempotent(t *testing.T) {
	env := setupEnv(t)

	orderID := uuid.NewString()
	storeID := uuid.NewString()
	today := time.Now().UTC().Format("2006-01-02")

	payload := map[string]any{
		"order_id": orderID, "store_id": storeID, "customer_id": uuid.NewString(),
		"fulfillment": "PICKUP", "payment_method": "WALLET",
		"items_total": int64(80000), "ship_fee": int64(0),
		"discount": int64(0), "grand_total": int64(80000),
		"items": []map[string]any{{"date": today}},
	}
	raw, _ := json.Marshal(payload)
	env2 := outbox.Envelope{EventID: uuid.NewString(), EventType: "order.placed", OccurredAt: time.Now().UTC(), Data: raw}
	b, _ := json.Marshal(env2)
	msg := kafka.Message{Value: b}

	_ = env.orderHandler.HandleKafkaMessage(context.Background(), msg)
	_ = env.orderHandler.HandleKafkaMessage(context.Background(), msg) // replay

	var cnt int
	env.db.Raw("SELECT orders_count FROM revenue_daily WHERE store_id=?", storeID).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("idempotent: expected orders_count=1 after replay, got %d", cnt)
	}
}

// TestOrderCompleted_UpdatesStatus verifies order.completed transitions status.
func TestOrderCompleted_UpdatesStatus(t *testing.T) {
	env := setupEnv(t)

	orderID := uuid.NewString()
	storeID := uuid.NewString()
	today := time.Now().UTC().Format("2006-01-02")

	placed := buildMsg("order.placed", map[string]any{
		"order_id": orderID, "store_id": storeID, "customer_id": uuid.NewString(),
		"fulfillment": "DELIVERY", "payment_method": "COD",
		"items_total": int64(100000), "ship_fee": int64(15000),
		"discount": int64(0), "grand_total": int64(115000),
		"items": []map[string]any{{"date": today}},
	})
	_ = env.orderHandler.HandleKafkaMessage(context.Background(), placed)

	completed := buildMsg("order.completed", map[string]any{"order_id": orderID})
	if err := env.orderHandler.HandleKafkaMessage(context.Background(), completed); err != nil {
		t.Fatalf("order.completed: %v", err)
	}

	var status string
	env.db.Raw("SELECT status FROM order_facts WHERE order_id=?", orderID).Scan(&status)
	if status != "COMPLETED" {
		t.Errorf("expected status=COMPLETED, got %q", status)
	}
}

// TestOrderCancelled_IncrementsCancelledCount verifies cancelled_count is bumped.
func TestOrderCancelled_IncrementsCancelledCount(t *testing.T) {
	env := setupEnv(t)

	orderID := uuid.NewString()
	storeID := uuid.NewString()
	today := time.Now().UTC().Format("2006-01-02")

	placed := buildMsg("order.placed", map[string]any{
		"order_id": orderID, "store_id": storeID, "customer_id": uuid.NewString(),
		"fulfillment": "DELIVERY", "payment_method": "COD",
		"items_total": int64(90000), "ship_fee": int64(10000),
		"discount": int64(0), "grand_total": int64(100000),
		"items": []map[string]any{{"date": today}},
	})
	_ = env.orderHandler.HandleKafkaMessage(context.Background(), placed)

	cancelled := buildMsg("order.cancelled", map[string]any{
		"order_id": orderID, "store_id": storeID, "customer_id": uuid.NewString(),
	})
	if err := env.orderHandler.HandleKafkaMessage(context.Background(), cancelled); err != nil {
		t.Fatalf("order.cancelled: %v", err)
	}

	var cancelledCnt int
	env.db.Raw("SELECT cancelled_count FROM revenue_daily WHERE store_id=?", storeID).Scan(&cancelledCnt)
	if cancelledCnt != 1 {
		t.Errorf("expected cancelled_count=1, got %d", cancelledCnt)
	}
}

// TestPayoutSettled_MarksOrdersSettled verifies payout.settled sets settled=true.
func TestPayoutSettled_MarksOrdersSettled(t *testing.T) {
	env := setupEnv(t)

	orderID := uuid.NewString()
	storeID := uuid.NewString()
	today := time.Now().UTC().Format("2006-01-02")

	placed := buildMsg("order.placed", map[string]any{
		"order_id": orderID, "store_id": storeID, "customer_id": uuid.NewString(),
		"fulfillment": "DELIVERY", "payment_method": "COD",
		"items_total": int64(60000), "ship_fee": int64(10000),
		"discount": int64(0), "grand_total": int64(70000),
		"items": []map[string]any{{"date": today}},
	})
	_ = env.orderHandler.HandleKafkaMessage(context.Background(), placed)

	completed := buildMsg("order.completed", map[string]any{"order_id": orderID})
	_ = env.orderHandler.HandleKafkaMessage(context.Background(), completed)

	settled := buildMsg("payout.settled", map[string]any{
		"store_id": storeID, "batch_id": uuid.NewString(), "amount": int64(70000),
	})
	if err := env.payHandler.HandlePayoutKafkaMessage(context.Background(), settled); err != nil {
		t.Fatalf("payout.settled: %v", err)
	}

	var isSettled bool
	env.db.Raw("SELECT settled FROM order_facts WHERE order_id=?", orderID).Scan(&isSettled)
	if !isSettled {
		t.Error("expected order_fact.settled=true after payout.settled")
	}
}

// TestReviewCreated_UpdatesStorePerformance verifies rating aggregates are updated.
func TestReviewCreated_UpdatesStorePerformance(t *testing.T) {
	env := setupEnv(t)
	storeID := uuid.NewString()

	for _, rating := range []int{4, 5, 3} {
		msg := buildMsg("review.created", map[string]any{
			"review_id": uuid.NewString(),
			"store_id":  storeID,
			"order_id":  uuid.NewString(),
			"rating":    rating,
		})
		if err := env.rvHandler.HandleReviewKafkaMessage(context.Background(), msg); err != nil {
			t.Fatalf("review.created: %v", err)
		}
	}

	var ratingCount int
	var avgRating float64
	env.db.Raw("SELECT rating_count, avg_rating FROM store_performance WHERE store_id=?", storeID).
		Row().Scan(&ratingCount, &avgRating)

	if ratingCount != 3 {
		t.Errorf("expected rating_count=3, got %d", ratingCount)
	}
	// avg of 4+5+3 = 12/3 = 4.00
	if avgRating < 3.99 || avgRating > 4.01 {
		t.Errorf("expected avg_rating≈4.00, got %.2f", avgRating)
	}
}

// TestVendorApproved_IncrementsNewVendors verifies user_growth_daily is updated.
func TestVendorApproved_IncrementsNewVendors(t *testing.T) {
	env := setupEnv(t)

	msg := buildMsg("vendor.approved", map[string]any{
		"vendor_id": uuid.NewString(),
	})
	if err := env.rvHandler.HandleVendorKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("vendor.approved: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	var newVendors int
	env.db.Raw("SELECT new_vendors FROM user_growth_daily WHERE date=?", today).Scan(&newVendors)
	if newVendors != 1 {
		t.Errorf("expected new_vendors=1, got %d", newVendors)
	}
}

// TestAnalyticsEndpoint_ReturnsAggregates seeds order_facts and verifies the
// HTTP analytics endpoint returns matching totals.
func TestAnalyticsEndpoint_ReturnsAggregates(t *testing.T) {
	env := setupEnv(t)

	storeID := uuid.NewString()
	now := time.Now().UTC()

	// Seed two completed orders directly into order_facts.
	env.db.Exec(`INSERT INTO order_facts
		(order_id, store_id, customer_id, date, fulfillment, items_total, ship_fee,
		 discount, grand_total, payment_method, status, settled, created_at, updated_at)
		VALUES
		(gen_random_uuid(), $1, gen_random_uuid(), $2, 'DELIVERY', 100000, 20000, 0, 120000, 'COD', 'COMPLETED', false, $3, $3),
		(gen_random_uuid(), $1, gen_random_uuid(), $2, 'PICKUP',   80000,  0,     0,  80000, 'WALLET', 'COMPLETED', false, $3, $3)`,
		storeID, now.Format("2006-01-02"), now)

	w := env.get(t, "/api/v1/admin/analytics?period=day")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			OrdersTotal  int64 `json:"OrdersTotal"`
			RevenueTotal int64 `json:"RevenueTotal"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Data.OrdersTotal != 2 {
		t.Errorf("expected OrdersTotal=2, got %d", resp.Data.OrdersTotal)
	}
	if resp.Data.RevenueTotal != 200000 {
		t.Errorf("expected RevenueTotal=200000, got %d", resp.Data.RevenueTotal)
	}
}

// TestRecentOrders_ReturnsRows verifies GET /admin/orders/recent returns rows.
func TestRecentOrders_ReturnsRows(t *testing.T) {
	env := setupEnv(t)

	now := time.Now().UTC()
	env.db.Exec(`INSERT INTO order_facts
		(order_id, store_id, customer_id, date, fulfillment, items_total, ship_fee,
		 discount, grand_total, payment_method, status, settled, created_at, updated_at)
		VALUES (gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), $1, 'DELIVERY',
		        50000, 10000, 0, 60000, 'COD', 'PENDING', false, $2, $2)`,
		now.Format("2006-01-02"), now)

	w := env.get(t, "/api/v1/admin/orders/recent?limit=5")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Error("expected at least one recent order, got none")
	}
}
