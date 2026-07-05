// Package integration covers the order service end-to-end against a running
// Postgres (order_db_test). Set SKIP_INTEGRATION=1 to skip.
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
	"testing"
	"time"

	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	authstore "project/pkg/auth/store"
	"project/pkg/config"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/audit"
	"project/pkg/outbox"
	"project/pkg/testutil"
	orderevent "project/services/order/internal/handler/event"
	handlerhttp "project/services/order/internal/handler/http"
	v1 "project/services/order/internal/handler/http/v1"
	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/infrastructure/persistence"
	"project/services/order/internal/repository"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ── path helpers ──────────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/order.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// ── test constants ────────────────────────────────────────────────────────────

const (
	testCustomerID = "11111111-1111-1111-1111-111111111111"
	testStoreID    = "bbbbbbbb-2222-2222-2222-222222222222"
	testItemID     = "aaaaaaaa-1111-1111-1111-111111111111"
	testLocationID = "eeeeeeee-5555-5555-5555-555555555555"
	testShipperID  = "dddddddd-4444-4444-4444-444444444444"
)

var allTables = []string{
	"order_status_history", "order_items", "orders",
	"cart_items", "carts",
	"outbox_events", "processed_events", "pending_compensations",
}

// ── store stub facade ─────────────────────────────────────────────────────────

type storeGRPCFacade interface {
	GetStoreForOrder(ctx context.Context, storeID, roomID, locationLevel string) (*grpcclient.StoreForOrderResult, error)
}

type promotionGRPCFacade interface {
	ApplyPromotion(ctx context.Context, orderID, storeID, customerID string, codes []string, subtotal int64, itemCount int32) (*grpcclient.PromotionApplyResult, error)
	ConfirmUsage(ctx context.Context, orderID string) error
	ReleaseUsage(ctx context.Context, orderID string) error
}

type paymentGRPCFacade interface {
	Capture(ctx context.Context, orderID, customerID string, amount int64, method string) (*grpcclient.PaymentCaptureResult, error)
	Refund(ctx context.Context, orderID string, amount int64) error
}

// ── stub implementations ──────────────────────────────────────────────────────

// stubStore returns an open store by default; set closed=true to simulate closed.
type stubStore struct{ closed bool }

func (s *stubStore) GetStoreForOrder(_ context.Context, _, _, _ string) (*grpcclient.StoreForOrderResult, error) {
	if s.closed {
		return &grpcclient.StoreForOrderResult{
			Found:          true,
			SaleStatus:     "PAUSED",
			OpenNow:        false,
			CloseTimeToday: "22:00",
		}, nil
	}
	return &grpcclient.StoreForOrderResult{
		Found:          true,
		SaleStatus:     "OPEN",
		UnitShipFee:    10000,
		Served:         true,
		OpenNow:        true,
		PrepMinutes:    15,
		OpenTimeToday:  "08:00",
		CloseTimeToday: "23:00",
		Items:          []grpcclient.StoreOrderItem{{ItemID: testItemID, Name: "Pho", Price: 50000}},
	}, nil
}

// GetStoreOwnership lets stubStore satisfy the owner handler's resolver — the
// owner-flow tests exercise lifecycle transitions, not authorization.
func (s *stubStore) GetStoreOwnership(_ context.Context, _ string) (*grpcclient.StoreOwnership, error) {
	return &grpcclient.StoreOwnership{Found: true, VendorID: "test-vendor", OwnerUserID: testCustomerID}, nil
}

// stubChecker authorizes every owner action so lifecycle tests stay focused.
type stubChecker struct{}

func (stubChecker) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (stubChecker) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

type stubPromotion struct{}

func (s *stubPromotion) ApplyPromotion(_ context.Context, _, _, _ string, _ []string, _ int64, _ int32) (*grpcclient.PromotionApplyResult, error) {
	return &grpcclient.PromotionApplyResult{Success: true}, nil
}
func (s *stubPromotion) ConfirmUsage(_ context.Context, _ string) error { return nil }
func (s *stubPromotion) ReleaseUsage(_ context.Context, _ string) error { return nil }

type stubPayment struct{}

func (s *stubPayment) Capture(_ context.Context, _, _ string, _ int64, _ string) (*grpcclient.PaymentCaptureResult, error) {
	return &grpcclient.PaymentCaptureResult{Status: "PENDING"}, nil
}
func (s *stubPayment) Refund(_ context.Context, _ string, _ int64) error { return nil }

// ── stub-aware PlaceOrderUsecase ──────────────────────────────────────────────
// PlaceOrderUsecase accepts *grpcclient.StoreClient etc. (concrete types).
// For tests we build a wrapper that satisfies the same PlaceOrderUsecase interface
// but uses facade interfaces, letting us inject stubs cleanly.

type testPlaceOrderUsecase struct {
	db         *gorm.DB
	orderRepo  repository.OrderRepository
	outboxRepo repository.OutboxRepository
	store      storeGRPCFacade
	promo      promotionGRPCFacade
	payment    paymentGRPCFacade
}

func (u *testPlaceOrderUsecase) PlaceOrder(ctx context.Context, req usecase.PlaceOrderRequest) (*usecase.PlaceOrderResult, error) {
	storeInfo, err := u.store.GetStoreForOrder(ctx, req.StoreID, req.LocationID, req.LocationLevel)
	if err != nil || !storeInfo.Found {
		return nil, usecase.ErrStoreNotFound
	}
	if !storeInfo.OpenNow {
		return nil, usecase.ErrStoreClosed
	}

	// Validate desired_time when provided.
	var desiredTimePtr *time.Time
	if req.DesiredTime != "" {
		dt, parseErr := time.Parse(time.RFC3339, req.DesiredTime)
		if parseErr != nil {
			return nil, usecase.ErrDesiredTimePast
		}
		now := time.Now()
		// Must be today (local).
		ny, nm, nd := now.Date()
		dy, dm, dd := dt.In(now.Location()).Date()
		if dy != ny || dm != nm || dd != nd {
			return nil, usecase.ErrDesiredTimeNotToday
		}
		// Must not be in the past.
		if dt.Before(now) {
			return nil, usecase.ErrDesiredTimePast
		}
		// Must not exceed close time.
		if storeInfo.CloseTimeToday != "" {
			var ch, cm int
			if _, scanErr := fmt.Sscanf(storeInfo.CloseTimeToday, "%d:%d", &ch, &cm); scanErr == nil {
				close := time.Date(now.Year(), now.Month(), now.Day(), ch, cm, 0, 0, now.Location())
				if dt.After(close) {
					return nil, usecase.ErrDesiredTimeAfterClose
				}
			}
		}
		desiredTimePtr = &dt
	}

	priceMap := make(map[string]grpcclient.StoreOrderItem, len(storeInfo.Items))
	for _, it := range storeInfo.Items {
		priceMap[it.ItemID] = it
	}

	var itemsTotal int64
	type itemRow struct {
		MenuItemID    string
		NameSnapshot  string
		PriceSnapshot int64
		Qty           int
	}
	var orderItems []itemRow
	for _, ri := range req.Items {
		si, ok := priceMap[ri.MenuItemID]
		if !ok {
			return nil, usecase.ErrItemNotInStore
		}
		itemsTotal += si.Price * int64(ri.Qty)
		orderItems = append(orderItems, itemRow{ri.MenuItemID, si.Name, si.Price, ri.Qty})
	}

	var shipFee int64
	if req.Fulfillment == "DELIVERY" {
		var totalQty int64
		for _, ri := range req.Items {
			totalQty += int64(ri.Qty)
		}
		shipFee = storeInfo.UnitShipFee * totalQty
	}

	grandTotal := itemsTotal + shipFee
	orderID := uuid.NewString()
	code := "VLX-TEST-" + orderID[:4]

	desiredTimeStr := ""
	if desiredTimePtr != nil {
		desiredTimeStr = desiredTimePtr.UTC().Format(time.RFC3339)
	}

	payload, _ := json.Marshal(map[string]any{
		"order_id":       orderID,
		"code":           code,
		"customer_id":    req.CustomerID,
		"store_id":       req.StoreID,
		"fulfillment":    string(req.Fulfillment),
		"payment_method": string(req.PaymentMethod),
		"items_total":    itemsTotal,
		"ship_fee":       shipFee,
		"discount":       int64(0),
		"grand_total":    grandTotal,
		"desired_time":   desiredTimeStr,
		"items":          orderItems,
	})

	orderRow := map[string]any{
		"id":             orderID,
		"code":           code,
		"customer_id":    req.CustomerID,
		"store_id":       req.StoreID,
		"fulfillment":    string(req.Fulfillment),
		"status":         "PENDING",
		"items_total":    itemsTotal,
		"ship_fee":       shipFee,
		"discount":       0,
		"grand_total":    grandTotal,
		"payment_method": string(req.PaymentMethod),
		"payment_status": "UNPAID",
		"voucher_codes":  json.RawMessage("[]"),
		"placed_at":      time.Now().UTC(),
	}
	if desiredTimePtr != nil {
		orderRow["desired_time"] = *desiredTimePtr
	}
	if req.LocationID != "" && req.Fulfillment == "DELIVERY" {
		orderRow["location_id"] = req.LocationID
	}

	return &usecase.PlaceOrderResult{
		OrderID:    orderID,
		Code:       code,
		GrandTotal: grandTotal,
	}, u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("orders").Create(orderRow).Error; err != nil {
			return err
		}
		for _, oi := range orderItems {
			if err := tx.Table("order_items").Create(map[string]any{
				"order_id":         orderID,
				"menu_item_id":     oi.MenuItemID,
				"name_snapshot":    oi.NameSnapshot,
				"price_snapshot":   oi.PriceSnapshot,
				"qty":              oi.Qty,
				"options_snapshot": json.RawMessage("[]"),
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Table("order_status_history").Create(map[string]any{
			"order_id": orderID,
			"status":   "PENDING",
		}).Error; err != nil {
			return err
		}
		return tx.Table("outbox_events").Create(map[string]any{
			"aggregate_type": "order",
			"aggregate_id":   orderID,
			"event_type":     "order.placed",
			"payload":        payload,
			"status":         "pending",
		}).Error
	})
}

// ── test environment ──────────────────────────────────────────────────────────

type testEnv struct {
	engine        *gin.Engine
	db            *gorm.DB
	cfg           *config.Config
	token         string
	customerID    string
	orderRepo     repository.OrderRepository
	outboxRepo    repository.OutboxRepository
	processedRepo repository.ProcessedEventRepository
	storeStub     *stubStore
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

	orderRepo := persistence.NewOrderGormRepository(db)
	cartRepo := persistence.NewCartGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)
	processedRepo := persistence.NewProcessedEventGormRepository(db)

	cartUC := usecase.NewCartUsecase(cartRepo)
	storeStub := &stubStore{}
	placeOrderUC := &testPlaceOrderUsecase{
		db:         db,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
		store:      storeStub,
		promo:      &stubPromotion{},
		payment:    &stubPayment{},
	}
	lifecycleUC := usecase.NewOrderLifecycleUsecase(db, orderRepo, cartRepo, outboxRepo, nil, nil, nil, audit.NoopLogger{})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), testCustomerID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		AuthMiddleware:  func(c *gin.Context) { c.Next() },
		CartHandler:     v1.NewCustomerCartHandler(cartUC),
		CustomerHandler: v1.NewCustomerOrderHandler(placeOrderUC, lifecycleUC),
		OwnerHandler:    v1.NewOwnerOrderHandler(lifecycleUC, nil, nil, storeStub, stubChecker{}),
	})

	token := "test-token"
	authStore, redisErr := authstore.NewRedisAuthStore(cfg.Redis)
	if redisErr == nil {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
		if pair, issueErr := jwtSvc.Issue(context.Background(), testCustomerID); issueErr == nil {
			_ = authStore.Whitelist(context.Background(), pair.JTIAccess, testCustomerID, pair.AccessTTL)
			token = pair.Access
		}
	}

	return &testEnv{
		engine:        engine,
		db:            db,
		cfg:           cfg,
		token:         token,
		customerID:    testCustomerID,
		orderRepo:     orderRepo,
		outboxRepo:    outboxRepo,
		processedRepo: processedRepo,
		storeStub:     storeStub,
	}
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

func buildKafkaMsg(eventType string, data json.RawMessage) kafka.Message {
	env := outbox.Envelope{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Data:       data,
	}
	b, _ := json.Marshal(env)
	return kafka.Message{Topic: "test", Value: b}
}

// ── Tests: Cart ───────────────────────────────────────────────────────────────

func TestCart_AddItem_ReturnsCart(t *testing.T) {
	env := setupEnv(t)
	body := `{"store_id":"` + testStoreID + `","menu_item_id":"` + testItemID + `","name_snapshot":"Pho","price_snapshot":50000,"qty":2}`
	w := env.do(t, "PUT", "/api/v1/me/cart/items", body)
	if w.Code != http.StatusOK {
		t.Fatalf("add cart item: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestCart_GetCart_Empty(t *testing.T) {
	env := setupEnv(t)
	w := env.do(t, "GET", "/api/v1/me/cart?store_id="+testStoreID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get cart: expected 200, got %d", w.Code)
	}
}

// A customer may hold items from several stores at once; the per-store carts
// coexist and GET /me/carts returns all of them.
func TestCart_MultiStore_Allowed(t *testing.T) {
	env := setupEnv(t)
	other := "cccccccc-3333-3333-3333-333333333333"
	w1 := env.do(t, "PUT", "/api/v1/me/cart/items",
		`{"store_id":"`+testStoreID+`","menu_item_id":"`+testItemID+`","name_snapshot":"Pho","price_snapshot":50000,"qty":1}`)
	if w1.Code != http.StatusOK {
		t.Fatalf("add store A: expected 200, got %d — %s", w1.Code, w1.Body.String())
	}
	w2 := env.do(t, "PUT", "/api/v1/me/cart/items",
		`{"store_id":"`+other+`","menu_item_id":"`+testItemID+`","name_snapshot":"Com","price_snapshot":40000,"qty":1}`)
	if w2.Code != http.StatusOK {
		t.Fatalf("add store B (cross-store): expected 200, got %d — %s", w2.Code, w2.Body.String())
	}

	w := env.do(t, "GET", "/api/v1/me/carts", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list carts: expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Carts []struct {
				StoreID string `json:"StoreID"`
			} `json:"carts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse carts: %v", err)
	}
	stores := map[string]bool{}
	for _, c := range resp.Data.Carts {
		stores[c.StoreID] = true
	}
	if !stores[testStoreID] || !stores[other] {
		t.Errorf("carts must contain both stores, got %d carts: %s", len(resp.Data.Carts), w.Body.String())
	}
}

// ── Tests: Place Order ────────────────────────────────────────────────────────

func TestPlaceOrder_COD_Delivery_HappyPath(t *testing.T) {
	env := setupEnv(t)
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", ""))
	if w.Code != http.StatusCreated {
		t.Fatalf("place order COD DELIVERY: expected 201, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			OrderID    string `json:"order_id"`
			Code       string `json:"code"`
			GrandTotal int64  `json:"grand_total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Data.OrderID == "" {
		t.Error("order_id must be non-empty")
	}
	// 1×50000 + 1×10000 ship = 60000
	if resp.Data.GrandTotal != 60000 {
		t.Errorf("grand_total: want 60000, got %d", resp.Data.GrandTotal)
	}

	var outboxCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='order.placed'").Scan(&outboxCount)
	if outboxCount != 1 {
		t.Errorf("want 1 order.placed outbox row, got %d", outboxCount)
	}
}

func TestPlaceOrder_PICKUP_ZeroShipFee(t *testing.T) {
	env := setupEnv(t)
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "PICKUP", ""))
	if w.Code != http.StatusCreated {
		t.Fatalf("place order PICKUP: expected 201, got %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct{ GrandTotal int64 `json:"grand_total"` } `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.GrandTotal != 50000 {
		t.Errorf("PICKUP grand_total: want 50000 (no ship), got %d", resp.Data.GrandTotal)
	}
}

func TestPlaceOrder_WithDesiredTime_Persisted(t *testing.T) {
	env := setupEnv(t)
	desired := desiredTimeInOneHourOrSkip(t)
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", desired))
	if w.Code != http.StatusCreated {
		t.Fatalf("desired_time happy path: expected 201, got %d — %s", w.Code, w.Body.String())
	}

	// Verify desired_time stored in outbox payload.
	var payload string
	env.db.Raw("SELECT payload FROM outbox_events WHERE event_type='order.placed' ORDER BY created_at DESC LIMIT 1").Scan(&payload)
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	dt, ok := m["desired_time"].(string)
	if !ok || dt == "" {
		t.Errorf("desired_time missing in order.placed payload, got %v", m["desired_time"])
	}
}

func TestPlaceOrder_DesiredTimeInPast_Rejected(t *testing.T) {
	env := setupEnv(t)
	// desired_time 2 hours in the past
	past := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", past))
	if w.Code != http.StatusBadRequest {
		t.Errorf("past desired_time: expected 400, got %d — %s", w.Code, w.Body.String())
	}
}

func TestPlaceOrder_DesiredTimeNotToday_Rejected(t *testing.T) {
	env := setupEnv(t)
	// desired_time = tomorrow
	tomorrow := time.Now().Add(25 * time.Hour).UTC().Format(time.RFC3339)
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", tomorrow))
	if w.Code != http.StatusBadRequest {
		t.Errorf("tomorrow desired_time: expected 400, got %d — %s", w.Code, w.Body.String())
	}
}

func TestPlaceOrder_StoreClosed_Rejected(t *testing.T) {
	env := setupEnv(t)
	env.storeStub.closed = true
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("closed store: expected 400, got %d — %s", w.Code, w.Body.String())
	}
}

func TestPlaceOrder_ASAP_NoDesiredTime(t *testing.T) {
	env := setupEnv(t)
	// No desired_time → ASAP; order must succeed and desired_time in payload = "".
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", ""))
	if w.Code != http.StatusCreated {
		t.Fatalf("ASAP order: expected 201, got %d — %s", w.Code, w.Body.String())
	}

	var payload string
	env.db.Raw("SELECT payload FROM outbox_events WHERE event_type='order.placed' ORDER BY created_at DESC LIMIT 1").Scan(&payload)
	var m map[string]any
	json.Unmarshal([]byte(payload), &m)
	dt, _ := m["desired_time"].(string)
	if dt != "" {
		t.Errorf("ASAP: desired_time in payload should be empty string, got %q", dt)
	}
}

// ── Tests: Lifecycle ──────────────────────────────────────────────────────────

func TestCancelOrder_PENDING_Succeeds(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	w := env.do(t, "POST", "/api/v1/orders/"+orderID+"/cancel", "")
	if w.Code != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var cancelCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='order.cancelled'").Scan(&cancelCount)
	if cancelCount != 1 {
		t.Errorf("want 1 order.cancelled outbox row, got %d", cancelCount)
	}
}

func TestCancelOrder_AfterCancelled_Fails(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	env.do(t, "POST", "/api/v1/orders/"+orderID+"/cancel", "")
	w := env.do(t, "POST", "/api/v1/orders/"+orderID+"/cancel", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("double-cancel: expected 400, got %d", w.Code)
	}
}

func TestAdvanceStatus_PENDING_to_CONFIRMED(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	w := env.do(t, "PATCH", "/api/v1/stores/"+testStoreID+"/orders/"+orderID+"/status",
		`{"status":"CONFIRMED"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("advance to CONFIRMED: expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var status string
	env.db.Raw("SELECT status FROM orders WHERE id=$1", orderID).Scan(&status)
	if status != "CONFIRMED" {
		t.Errorf("status: want CONFIRMED, got %s", status)
	}
}

// ── Tests: order.ready carries desired_time ───────────────────────────────────

func TestOrderReady_DesiredTimeInEvent(t *testing.T) {
	env := setupEnv(t)
	desired := desiredTimeInOneHourOrSkip(t)
	orderID := mustPlaceOrderWithDesiredTime(t, env, desired)

	// Advance to READY via CONFIRMED → PREPARING → READY.
	for _, s := range []string{"CONFIRMED", "PREPARING", "READY"} {
		w := env.do(t, "PATCH", "/api/v1/stores/"+testStoreID+"/orders/"+orderID+"/status",
			`{"status":"`+s+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("advance to %s: got %d — %s", s, w.Code, w.Body.String())
		}
	}

	var payload string
	env.db.Raw("SELECT payload FROM outbox_events WHERE event_type='order.ready' AND aggregate_id=$1", orderID).Scan(&payload)
	if payload == "" {
		t.Fatal("order.ready outbox event not found")
	}
	var m map[string]any
	json.Unmarshal([]byte(payload), &m)
	dt, _ := m["desired_time"].(string)
	if dt == "" {
		t.Errorf("desired_time missing in order.ready payload, got %v", m["desired_time"])
	}
}

// ── Tests: Event consumers ────────────────────────────────────────────────────

func TestPaymentCapturedEvent_MarksOrderPaid(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	handler := orderevent.NewPaymentEventHandler(env.db, env.orderRepo, env.processedRepo)
	data, _ := json.Marshal(map[string]any{"order_id": orderID})

	if err := handler.HandleKafkaMessage(context.Background(), buildKafkaMsg("payment.captured", data)); err != nil {
		t.Fatalf("payment.captured: %v", err)
	}

	var payStatus string
	env.db.Raw("SELECT payment_status FROM orders WHERE id=$1", orderID).Scan(&payStatus)
	if payStatus != "PAID" {
		t.Errorf("want PAID, got %s", payStatus)
	}

	// Idempotency
	if err := handler.HandleKafkaMessage(context.Background(), buildKafkaMsg("payment.captured", data)); err != nil {
		t.Errorf("idempotent: unexpected error: %v", err)
	}
	env.db.Raw("SELECT payment_status FROM orders WHERE id=$1", orderID).Scan(&payStatus)
	if payStatus != "PAID" {
		t.Errorf("idempotent: want PAID, got %s", payStatus)
	}
}

func TestDeliveryStatusChangedEvent_SyncsDelivering(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	// Manually advance to READY so the DELIVERING transition is valid.
	env.db.Exec("UPDATE orders SET status='READY' WHERE id=$1", orderID)

	handler := orderevent.NewDeliveryEventHandler(env.db, env.orderRepo, env.outboxRepo, env.processedRepo)
	data, _ := json.Marshal(map[string]any{
		"order_id":   orderID,
		"status":     "DELIVERING",
		"shipper_id": testShipperID,
	})

	if err := handler.HandleKafkaMessage(context.Background(), buildKafkaMsg("delivery.status_changed", data)); err != nil {
		t.Fatalf("delivery.status_changed: %v", err)
	}

	var status string
	env.db.Raw("SELECT status FROM orders WHERE id=$1", orderID).Scan(&status)
	if status != "DELIVERING" {
		t.Errorf("want DELIVERING, got %s", status)
	}
}

func TestDeliveryStatusChangedEvent_Delivered_PublishesOrderDelivered(t *testing.T) {
	env := setupEnv(t)
	orderID := mustPlaceOrder(t, env)

	env.db.Exec("UPDATE orders SET status='DELIVERING' WHERE id=$1", orderID)

	handler := orderevent.NewDeliveryEventHandler(env.db, env.orderRepo, env.outboxRepo, env.processedRepo)
	data, _ := json.Marshal(map[string]any{
		"order_id":   orderID,
		"status":     "DELIVERED",
		"shipper_id": testShipperID,
	})

	if err := handler.HandleKafkaMessage(context.Background(), buildKafkaMsg("delivery.status_changed", data)); err != nil {
		t.Fatalf("delivery.status_changed DELIVERED: %v", err)
	}

	var deliveredCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='order.delivered' AND aggregate_id=$1", orderID).Scan(&deliveredCount)
	if deliveredCount != 1 {
		t.Errorf("want 1 order.delivered outbox row, got %d", deliveredCount)
	}

	var completedCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='order.completed' AND aggregate_id=$1", orderID).Scan(&completedCount)
	if completedCount != 1 {
		t.Errorf("want 1 order.completed outbox row, got %d", completedCount)
	}
}

func TestProcessedEvents_Deduplication(t *testing.T) {
	env := setupEnv(t)
	eventID := uuid.NewString()

	inserted1, err := env.processedRepo.MarkProcessed(context.Background(), env.db, eventID)
	if err != nil || !inserted1 {
		t.Fatalf("first MarkProcessed: inserted=%v err=%v", inserted1, err)
	}
	inserted2, err := env.processedRepo.MarkProcessed(context.Background(), env.db, eventID)
	if err != nil || inserted2 {
		t.Errorf("second MarkProcessed: inserted=%v err=%v (want false, nil)", inserted2, err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildPlaceOrderBody(paymentMethod, fulfillment, desiredTime string) string {
	locPart := ""
	if fulfillment == "DELIVERY" {
		locPart = `"location_id":"` + testLocationID + `",`
	}
	dtPart := ""
	if desiredTime != "" {
		dtPart = `,"desired_time":"` + desiredTime + `"`
	}
	return `{"store_id":"` + testStoreID + `",` + locPart +
		`"fulfillment":"` + fulfillment + `",` +
		`"payment_method":"` + paymentMethod + `",` +
		`"items":[{"menu_item_id":"` + testItemID + `","qty":1}]` +
		dtPart + `}`
}

func mustPlaceOrder(t *testing.T, env *testEnv) string {
	t.Helper()
	return mustPlaceOrderWithDesiredTime(t, env, "")
}

// desiredTimeInOneHourOrSkip returns now+1h as the order's desired time.
// Desired times must fall on the same day and before the stub store's 23:00
// close, so when the suite runs after 22:00 local there is no valid value —
// skip instead of failing on the business rule.
func desiredTimeInOneHourOrSkip(t *testing.T) string {
	t.Helper()
	now := time.Now()
	dt := now.Add(time.Hour)
	closeAt := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())
	if dt.Day() != now.Day() || dt.After(closeAt) {
		t.Skip("no valid same-day desired_time this close to store close (23:00)")
	}
	return dt.UTC().Format(time.RFC3339)
}

func mustPlaceOrderWithDesiredTime(t *testing.T, env *testEnv, desiredTime string) string {
	t.Helper()
	w := env.do(t, "POST", "/api/v1/orders", buildPlaceOrderBody("COD", "DELIVERY", desiredTime))
	if w.Code != http.StatusCreated {
		t.Fatalf("mustPlaceOrder: expected 201, got %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct{ OrderID string `json:"order_id"` } `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("mustPlaceOrder: parse: %v", err)
	}
	return resp.Data.OrderID
}

// TestPendingCompensationLifecycle: a persisted rollback intent moves through
// open → failed(attempts++) → done, and drops out of ListOpen at each terminal
// state — the contract the compensation worker relies on.
func TestPendingCompensationLifecycle(t *testing.T) {
	env := setupEnv(t)
	repo := persistence.NewCompensationGormRepository(env.db)
	ctx := context.Background()

	row := &entity.PendingCompensation{
		OrderID: "3141cd6b-0000-0000-0000-000000000001",
		Action:  entity.CompensationRefund,
		Amount:  60000,
	}
	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("create: %v", err)
	}

	open, err := repo.ListOpen(ctx, 20, 10)
	if err != nil || len(open) != 1 {
		t.Fatalf("ListOpen after create: err=%v len=%d, want 1", err, len(open))
	}

	if err := repo.MarkFailed(ctx, open[0].ID, "payment unavailable"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	open, _ = repo.ListOpen(ctx, 20, 10)
	if len(open) != 1 || open[0].Attempts != 1 || open[0].LastError == nil {
		t.Fatalf("after MarkFailed: len=%d attempts=%d lastError=%v", len(open), open[0].Attempts, open[0].LastError)
	}

	// Attempt cap: rows at/over maxAttempts stop being listed.
	if rows, _ := repo.ListOpen(ctx, 1, 10); len(rows) != 0 {
		t.Errorf("ListOpen with maxAttempts=1: want 0 rows, got %d", len(rows))
	}

	if err := repo.MarkDone(ctx, open[0].ID); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}
	if rows, _ := repo.ListOpen(ctx, 20, 10); len(rows) != 0 {
		t.Errorf("ListOpen after done: want 0 rows, got %d", len(rows))
	}
}
