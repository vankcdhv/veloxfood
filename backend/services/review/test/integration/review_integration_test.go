// Package integration tests the review service end-to-end against review_db_test.
// Set SKIP_INTEGRATION=1 to skip when DB is unavailable.
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
	"project/pkg/testutil"
	grpchandler "project/services/review/internal/handler/grpc"
	handlerhttp "project/services/review/internal/handler/http"
	v1 "project/services/review/internal/handler/http/v1"
	"project/services/review/internal/infrastructure/grpcclient"
	"project/services/review/internal/infrastructure/persistence"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	reviewv1 "project/proto/review/v1"
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

// denyChecker denies every permission — used to test authz failures.
type denyChecker struct{}

func (denyChecker) HasPermission(context.Context, string, string) (bool, error) {
	return false, nil
}
func (denyChecker) HasVendorPermission(context.Context, string, string, string) (bool, error) {
	return false, nil
}

// completedOrderClient returns a COMPLETED order matching customerID.
type completedOrderClient struct {
	customerID string
	storeID    string
}

func (c *completedOrderClient) GetOrder(_ context.Context, orderID string) (*grpcclient.OrderResult, error) {
	return &grpcclient.OrderResult{
		Found:      true,
		OrderID:    orderID,
		CustomerID: c.customerID,
		StoreID:    c.storeID,
		Status:     "COMPLETED",
	}, nil
}

// pendingOrderClient returns an order that is NOT completed.
type pendingOrderClient struct {
	customerID string
	storeID    string
}

func (c *pendingOrderClient) GetOrder(_ context.Context, orderID string) (*grpcclient.OrderResult, error) {
	return &grpcclient.OrderResult{
		Found:      true,
		OrderID:    orderID,
		CustomerID: c.customerID,
		StoreID:    c.storeID,
		Status:     "PLACED",
	}, nil
}

// otherCustomerOrderClient returns an order belonging to a different customer.
type otherCustomerOrderClient struct {
	storeID string
}

func (c *otherCustomerOrderClient) GetOrder(_ context.Context, orderID string) (*grpcclient.OrderResult, error) {
	return &grpcclient.OrderResult{
		Found:      true,
		OrderID:    orderID,
		CustomerID: "other-customer-uuid",
		StoreID:    c.storeID,
		Status:     "COMPLETED",
	}, nil
}

// fixedStoreClient returns a known store ownership for any store_id.
type fixedStoreClient struct {
	vendorID    string
	ownerUserID string
}

func (c *fixedStoreClient) GetStoreOwnership(_ context.Context, _ string) (*grpcclient.StoreOwnership, error) {
	return &grpcclient.StoreOwnership{
		Found:       true,
		VendorID:    c.vendorID,
		OwnerUserID: c.ownerUserID,
	}, nil
}

// ── Test environment ─────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/review.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

type testEnv struct {
	engine     *gin.Engine
	db         *gorm.DB
	token      string
	userID     string
	storeID    string
	vendorID   string
	orderID    string
	grpcSrv    *grpchandler.ReviewServiceServer
}

func setupEnv(t *testing.T, orderClient grpcclient.OrderClient, checker authmw.PermissionChecker) *testEnv {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("SKIP_INTEGRATION=1")
	}

	cfg, err := config.Load(configPath())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db := testutil.SetupTestDB(t, cfg, migrationsPath())
	tables := []string{"review_reports", "review_replies", "reviews", "outbox_events", "processed_events"}
	testutil.Truncate(t, db, tables...)
	t.Cleanup(func() { testutil.Truncate(t, db, tables...) })

	userID := "00000000-0000-0000-0000-000000000001"
	storeID := "aaaaaaaa-0000-0000-0000-000000000001"
	vendorID := "bbbbbbbb-0000-0000-0000-000000000001"
	orderID := fmt.Sprintf("cccccccc-0000-0000-0000-%012d", 1)

	if orderClient == nil {
		orderClient = &completedOrderClient{customerID: userID, storeID: storeID}
	}
	if checker == nil {
		checker = allowChecker{}
	}

	storeClient := &fixedStoreClient{vendorID: vendorID, ownerUserID: userID}

	reviewRepo := persistence.NewReviewGormRepository(db)
	replyRepo := persistence.NewReviewReplyGormRepository(db)
	reportRepo := persistence.NewReviewReportGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)
	processedRepo := persistence.NewProcessedEventGormRepository(db)

	reviewUC := usecase.NewReviewUsecase(
		db, reviewRepo, replyRepo, reportRepo, outboxRepo, processedRepo,
		orderClient, storeClient, checker,
	)
	grpcSrv := grpchandler.NewReviewServiceServer(reviewUC)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	// Inject userID into context so handlers can read it.
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		// nil uploader: photo upload returns 503 in tests, no MinIO in the loop.
		CustomerHandler: v1.NewCustomerReviewHandler(reviewUC, nil),
		OwnerHandler:    v1.NewOwnerReviewHandler(reviewUC),
		AdminHandler:    v1.NewAdminReviewHandler(reviewUC),
		AuthMiddleware:  func(c *gin.Context) { c.Next() },
		PermChecker:     checker,
	})

	return &testEnv{
		engine:   engine,
		db:       db,
		userID:   userID,
		storeID:  storeID,
		vendorID: vendorID,
		orderID:  orderID,
		grpcSrv:  grpcSrv,
		token:    "test-token",
	}
}

func setup(t *testing.T) *testEnv {
	return setupEnv(t, nil, nil)
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

// issueToken generates a real JWT and whitelists it in Redis.
func issueToken(t *testing.T, cfg *config.Config, userID string) (string, bool) {
	t.Helper()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
	authStoreR, err := authstore.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return "", false
	}
	pair, err := jwtSvc.Issue(context.Background(), userID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	_ = authStoreR.Whitelist(context.Background(), pair.JTIAccess, userID, pair.AccessTTL)
	return pair.Access, true
}

// ── Happy path: create review ────────────────────────────────────────────────

func TestCreateReview_HappyPath(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{
		"target_type": "STORE",
		"target_id": "%s",
		"rating": 5,
		"comment": "Great food!"
	}`, env.storeID)

	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create review: %d — %s", w.Code, w.Body.String())
	}

	reviewID := dataID(t, w)
	if reviewID == "" {
		t.Fatal("expected review ID in response")
	}

	// Verify in DB.
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM reviews WHERE id = ? AND store_id = ? AND rating = 5", reviewID, env.storeID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 review in DB, got %d", count)
	}

	// Verify outbox row enqueued.
	var outboxCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type = 'review.created' AND aggregate_id = ?", reviewID).Scan(&outboxCount)
	if outboxCount != 1 {
		t.Errorf("expected review.created outbox row, got %d", outboxCount)
	}
}

// ── Reject non-completed order ───────────────────────────────────────────────

func TestCreateReview_RejectNonCompleted(t *testing.T) {
	env := setupEnv(t, &pendingOrderClient{
		customerID: "00000000-0000-0000-0000-000000000001",
		storeID:    "aaaaaaaa-0000-0000-0000-000000000001",
	}, nil)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":4,"comment":"ok"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code == http.StatusCreated {
		t.Fatal("expected non-201 for non-completed order")
	}
}

// ── Reject wrong customer ────────────────────────────────────────────────────

func TestCreateReview_RejectWrongCustomer(t *testing.T) {
	env := setupEnv(t, &otherCustomerOrderClient{
		storeID: "aaaaaaaa-0000-0000-0000-000000000001",
	}, nil)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":3,"comment":"meh"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code == http.StatusCreated {
		t.Fatal("expected non-201 for wrong customer")
	}
}

// ── Reject duplicate review ──────────────────────────────────────────────────

func TestCreateReview_RejectDuplicate(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":4,"comment":"first"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: %d — %s", w.Code, w.Body.String())
	}

	// Second attempt same order+target — must conflict.
	w2 := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate, got %d — %s", w2.Code, w2.Body.String())
	}
}

// ── List store reviews ───────────────────────────────────────────────────────

func TestListStoreReviews(t *testing.T) {
	env := setup(t)

	// Create a review first.
	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":5,"comment":"list test"}`, env.storeID)
	env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)

	w := env.do(t, "GET", "/api/v1/stores/"+env.storeID+"/reviews", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list reviews: %d — %s", w.Code, w.Body.String())
	}
	// Handler returns the list nested as data:{Items,Total} (matches the FE
	// ReviewListResponse contract consumed by store-reviews-list.tsx).
	var resp struct {
		Data struct {
			Items []struct{ ID string }
			Total int64
		}
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data.Items) < 1 {
		t.Errorf("expected at least 1 review, got %d", len(resp.Data.Items))
	}
}

// ── Owner reply ──────────────────────────────────────────────────────────────

func TestReplyToReview_OwnerAllowed(t *testing.T) {
	env := setup(t)

	// Create review.
	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":4,"comment":"nice"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	reviewID := dataID(t, w)

	replyBody := `{"content":"Thank you for your feedback!"}`
	w2 := env.do(t, "POST", "/api/v1/reviews/"+reviewID+"/reply", replyBody)
	if w2.Code != http.StatusCreated {
		t.Fatalf("reply: %d — %s", w2.Code, w2.Body.String())
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM review_replies WHERE review_id = ?", reviewID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 reply in DB, got %d", count)
	}
}

func TestReplyToReview_DeniedWhenNoPermission(t *testing.T) {
	// Use deny checker from the start so the review exists in the same DB,
	// but the reply permission check returns false.
	env := setupEnv(t, nil, denyChecker{})

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":3,"comment":"deny test"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create review: %d — %s", w.Code, w.Body.String())
	}
	reviewID := dataID(t, w)

	replyBody := `{"content":"Should be denied"}`
	w2 := env.do(t, "POST", "/api/v1/reviews/"+reviewID+"/reply", replyBody)
	if w2.Code == http.StatusCreated {
		t.Fatalf("expected non-201 for denied reply, got %d", w2.Code)
	}
}

// ── Report review ────────────────────────────────────────────────────────────

func TestReportReview(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":1,"comment":"bad"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	reviewID := dataID(t, w)

	reportBody := `{"reason":"inappropriate content"}`
	w2 := env.do(t, "POST", "/api/v1/reviews/"+reviewID+"/report", reportBody)
	if w2.Code != http.StatusCreated {
		t.Fatalf("report: %d — %s", w2.Code, w2.Body.String())
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM review_reports WHERE review_id = ?", reviewID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 report in DB, got %d", count)
	}

	// Verify review.reported outbox row.
	var outboxCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type = 'review.reported'").Scan(&outboxCount)
	if outboxCount != 1 {
		t.Errorf("expected review.reported outbox row, got %d", outboxCount)
	}
}

// ── Admin hide / restore ──────────────────────────────────────────────────────

func TestAdminHideAndRestoreReview(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":2,"comment":"hide me"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	reviewID := dataID(t, w)

	// Hide.
	wh := env.do(t, "PATCH", "/api/v1/admin/reviews/"+reviewID+"/hide", "")
	if wh.Code != http.StatusOK {
		t.Fatalf("hide: %d — %s", wh.Code, wh.Body.String())
	}
	var status string
	env.db.Raw("SELECT status FROM reviews WHERE id = ?", reviewID).Scan(&status)
	if status != "HIDDEN" {
		t.Errorf("expected HIDDEN, got %s", status)
	}

	// Restore.
	wr := env.do(t, "PATCH", "/api/v1/admin/reviews/"+reviewID+"/restore", "")
	if wr.Code != http.StatusOK {
		t.Fatalf("restore: %d — %s", wr.Code, wr.Body.String())
	}
	env.db.Raw("SELECT status FROM reviews WHERE id = ?", reviewID).Scan(&status)
	if status != "VISIBLE" {
		t.Errorf("expected VISIBLE, got %s", status)
	}
}

// ── Rating summary (gRPC) ─────────────────────────────────────────────────────

func TestGetStoreRatingSummary(t *testing.T) {
	env := setup(t)

	// Create two reviews with different ratings.
	body1 := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":4,"comment":"good"}`, env.storeID)
	env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body1)

	// Second review uses a different order ID to avoid duplicate violation.
	orderID2 := fmt.Sprintf("cccccccc-0000-0000-0000-%012d", 2)
	body2 := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":2,"comment":"meh"}`, env.storeID)
	env.do(t, "POST", "/api/v1/orders/"+orderID2+"/reviews", body2)

	resp, err := env.grpcSrv.GetStoreRatingSummary(context.Background(), &reviewv1.GetStoreRatingSummaryRequest{
		StoreId: env.storeID,
	})
	if err != nil {
		t.Fatalf("GetStoreRatingSummary: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected count=2, got %d", resp.Count)
	}
	// avg of 4 and 2 = 3.0
	if resp.Avg < 2.9 || resp.Avg > 3.1 {
		t.Errorf("expected avg≈3.0, got %.2f", resp.Avg)
	}
}

// ── review.created outbox row ─────────────────────────────────────────────────

func TestReviewCreatedOutboxRow(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":5,"comment":"outbox test"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d — %s", w.Code, w.Body.String())
	}
	reviewID := dataID(t, w)

	var payload string
	env.db.Raw(
		"SELECT payload::text FROM outbox_events WHERE event_type = 'review.created' AND aggregate_id = ?",
		reviewID,
	).Scan(&payload)

	if payload == "" {
		t.Fatal("expected outbox payload for review.created")
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		t.Fatalf("parse outbox payload: %v", err)
	}
	if p["review_id"] != reviewID {
		t.Errorf("expected review_id=%s in payload, got %v", reviewID, p["review_id"])
	}
}

// ── Idempotent event consumer ────────────────────────────────────────────────
// Tests that processed_events deduplication works by attempting to insert
// the same event_id twice; second insert should fail cleanly.

func TestProcessedEventIdempotency(t *testing.T) {
	env := setup(t)

	processedRepo := persistence.NewProcessedEventGormRepository(env.db)
	eventID := "test-event-id-idempotent-001"

	// First insert — should succeed.
	err := processedRepo.Insert(context.Background(), env.db, eventID)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Second insert — should fail with unique constraint violation.
	err2 := processedRepo.Insert(context.Background(), env.db, eventID)
	if err2 == nil {
		t.Error("expected error on duplicate event_id insert, got nil")
	}
}

// ── Invalid rating ───────────────────────────────────────────────────────────

func TestCreateReview_InvalidRating(t *testing.T) {
	env := setup(t)
	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":6,"comment":"bad rating"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	if w.Code == http.StatusCreated {
		t.Fatal("expected rejection for rating=6")
	}
}

// ── Admin list reported reviews ───────────────────────────────────────────────

func TestAdminListReportedReviews(t *testing.T) {
	env := setup(t)

	body := fmt.Sprintf(`{"target_type":"STORE","target_id":"%s","rating":1,"comment":"bad stuff"}`, env.storeID)
	w := env.do(t, "POST", "/api/v1/orders/"+env.orderID+"/reviews", body)
	reviewID := dataID(t, w)

	env.do(t, "POST", "/api/v1/reviews/"+reviewID+"/report", `{"reason":"spam"}`)

	w2 := env.do(t, "GET", "/api/v1/admin/reviews/reported", "")
	if w2.Code != http.StatusOK {
		t.Fatalf("list reported: %d — %s", w2.Code, w2.Body.String())
	}
	var resp struct {
		Data struct {
			Items []struct{ ID string } `json:"items"`
			Total int64                 `json:"total"`
			Page  int                   `json:"page"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp.Data.Total < 1 {
		t.Errorf("expected at least 1 reported review, got total=%d", resp.Data.Total)
	}
}
