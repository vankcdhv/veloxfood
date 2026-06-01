// Package integration covers the notification service end-to-end against a
// running Postgres (notification_db_test). Set SKIP_INTEGRATION=1 to skip.
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
	authstore "project/pkg/auth/store"
	"project/pkg/config"
	pkgfirebase "project/pkg/firebase"
	pkgmiddleware "project/pkg/middleware"
	"project/pkg/outbox"
	"project/pkg/testutil"
	notifEvent "project/services/notification/internal/handler/event"
	handlerhttp "project/services/notification/internal/handler/http"
	v1 "project/services/notification/internal/handler/http/v1"
	"project/services/notification/internal/infrastructure/persistence"
	"project/services/notification/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ── path helpers ──────────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/notification.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// ── constants ─────────────────────────────────────────────────────────────────

const (
	testCustomerID = "11111111-1111-1111-1111-111111111111"
	testOwnerID    = "22222222-2222-2222-2222-222222222222"
	testOrderID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testOrderCode  = "VLX-0001"
)

var allTables = []string{"notifications", "device_tokens", "processed_events"}

// ── test environment ──────────────────────────────────────────────────────────

type testEnv struct {
	engine  *gin.Engine
	token   string
	uc      *usecase.NotificationUsecase
	orderH  *notifEvent.OrderEventHandler
	payH    *notifEvent.PaymentEventHandler
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

	notifRepo := persistence.NewNotificationGormRepository(db)
	tokenRepo := persistence.NewDeviceTokenGormRepository(db)
	processedRepo := persistence.NewProcessedEventGormRepository(db)

	// Firebase disabled for tests — no real creds needed.
	fbClient, _ := pkgfirebase.New(config.FirebaseConfig{})
	uc := usecase.NewNotificationUsecase(db, notifRepo, tokenRepo, processedRepo, fbClient, nil)

	orderH := notifEvent.NewOrderEventHandler(uc)
	payH := notifEvent.NewPaymentEventHandler(uc)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), testCustomerID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		AuthMiddleware: func(c *gin.Context) { c.Next() },
		Handler:        v1.NewNotificationHandler(uc),
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
		engine:  engine,
		token:   token,
		uc:      uc,
		orderH:  orderH,
		payH:    payH,
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

func buildKafkaMsg(eventType string, data any) kafka.Message {
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

// ── Tests: Event consumers ────────────────────────────────────────────────────

// TestOrderConfirmed_CreatesInAppNotification verifies that consuming an
// order.confirmed event creates one in-app notification for the customer.
func TestOrderConfirmed_CreatesInAppNotification(t *testing.T) {
	env := setupEnv(t)

	msg := buildKafkaMsg("order.confirmed", map[string]any{
		"order_id":    testOrderID,
		"code":        testOrderCode,
		"customer_id": testCustomerID,
	})

	if err := env.orderH.HandleKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("HandleKafkaMessage: %v", err)
	}

	// Verify notification row was created.
	result, err := env.uc.List(context.Background(), testCustomerID, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("want 1 notification, got %d", result.Total)
	}
	if result.Items[0].Type != "order.confirmed" {
		t.Errorf("type: want order.confirmed, got %s", result.Items[0].Type)
	}
}

// TestOrderConfirmed_IdempotentReplay verifies that replaying the same event
// does not create a duplicate notification.
func TestOrderConfirmed_IdempotentReplay(t *testing.T) {
	env := setupEnv(t)

	eventID := uuid.NewString()
	payload := map[string]any{
		"order_id":    testOrderID,
		"code":        testOrderCode,
		"customer_id": testCustomerID,
	}
	raw, _ := json.Marshal(payload)
	env1 := outbox.Envelope{EventID: eventID, EventType: "order.confirmed", OccurredAt: time.Now().UTC(), Data: raw}
	b1, _ := json.Marshal(env1)
	msg := kafka.Message{Topic: "test", Value: b1}

	// First delivery.
	if err := env.orderH.HandleKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	// Replay same message.
	if err := env.orderH.HandleKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("replay: %v", err)
	}

	result, err := env.uc.List(context.Background(), testCustomerID, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("idempotent: want 1 notification, got %d", result.Total)
	}
}

// TestUnreadCount_AfterNotification verifies unread-count increments.
func TestUnreadCount_AfterNotification(t *testing.T) {
	env := setupEnv(t)

	// Inject two notifications via event handler.
	for _, code := range []string{"VLX-0001", "VLX-0002"} {
		msg := buildKafkaMsg("order.confirmed", map[string]any{
			"order_id":    uuid.NewString(),
			"code":        code,
			"customer_id": testCustomerID,
		})
		if err := env.orderH.HandleKafkaMessage(context.Background(), msg); err != nil {
			t.Fatalf("inject notification: %v", err)
		}
	}

	count, err := env.uc.UnreadCount(context.Background(), testCustomerID)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 2 {
		t.Errorf("unread count: want 2, got %d", count)
	}
}

// TestMarkRead_SingleNotification verifies PATCH .../read marks one notification.
func TestMarkRead_SingleNotification(t *testing.T) {
	env := setupEnv(t)

	msg := buildKafkaMsg("order.confirmed", map[string]any{
		"order_id": testOrderID, "code": testOrderCode, "customer_id": testCustomerID,
	})
	if err := env.orderH.HandleKafkaMessage(context.Background(), msg); err != nil {
		t.Fatalf("inject: %v", err)
	}

	result, _ := env.uc.List(context.Background(), testCustomerID, 0, 1)
	if result.Total == 0 {
		t.Fatal("no notifications to mark read")
	}
	notifID := result.Items[0].ID

	w := env.do(t, "PATCH", "/api/v1/me/notifications/"+notifID+"/read", "")
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH read: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	count, _ := env.uc.UnreadCount(context.Background(), testCustomerID)
	if count != 0 {
		t.Errorf("after mark-read: want 0 unread, got %d", count)
	}
}

// TestMarkAllRead marks all unread notifications at once.
func TestMarkAllRead(t *testing.T) {
	env := setupEnv(t)

	for i := 0; i < 3; i++ {
		msg := buildKafkaMsg("order.confirmed", map[string]any{
			"order_id": uuid.NewString(), "code": "VLX-TEST", "customer_id": testCustomerID,
		})
		_ = env.orderH.HandleKafkaMessage(context.Background(), msg)
	}

	w := env.do(t, "POST", "/api/v1/me/notifications/read-all", "")
	if w.Code != http.StatusOK {
		t.Fatalf("read-all: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	count, _ := env.uc.UnreadCount(context.Background(), testCustomerID)
	if count != 0 {
		t.Errorf("after read-all: want 0, got %d", count)
	}
}

// TestListNotifications_Pagination verifies GET /me/notifications returns correct shape.
func TestListNotifications_Pagination(t *testing.T) {
	env := setupEnv(t)

	for i := 0; i < 5; i++ {
		msg := buildKafkaMsg("order.confirmed", map[string]any{
			"order_id": uuid.NewString(), "code": "VLX-TEST", "customer_id": testCustomerID,
		})
		_ = env.orderH.HandleKafkaMessage(context.Background(), msg)
	}

	w := env.do(t, "GET", "/api/v1/me/notifications?skip=0&limit=3", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Items []any `json:"Items"`
			Total int64 `json:"Total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Data.Total != 5 {
		t.Errorf("total: want 5, got %d", resp.Data.Total)
	}
	if len(resp.Data.Items) != 3 {
		t.Errorf("page size: want 3, got %d", len(resp.Data.Items))
	}
}

// TestDeviceToken_RegisterAndDelete verifies the FCM token lifecycle.
func TestDeviceToken_RegisterAndDelete(t *testing.T) {
	env := setupEnv(t)

	// Register.
	regBody := `{"fcm_token":"test-fcm-token-abc123","platform":"web"}`
	w := env.do(t, "POST", "/api/v1/me/device-tokens", regBody)
	if w.Code != http.StatusOK {
		t.Fatalf("register token: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Idempotent re-register (same token, same user).
	w2 := env.do(t, "POST", "/api/v1/me/device-tokens", regBody)
	if w2.Code != http.StatusOK {
		t.Errorf("re-register: expected 200, got %d — %s", w2.Code, w2.Body.String())
	}

	// Delete by fetching token from DB directly (no list endpoint per spec).
	tokens, err := env.uc.ListTokensForUser(context.Background(), testCustomerID)
	if err != nil {
		t.Fatalf("ListTokensForUser: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatal("expected at least one token")
	}

	tokenID := tokens[0].ID
	w3 := env.do(t, "DELETE", "/api/v1/me/device-tokens/"+tokenID, "")
	if w3.Code != http.StatusOK {
		t.Fatalf("delete token: expected 200, got %d — %s", w3.Code, w3.Body.String())
	}
}

// TestFirebaseDisabled_NoError verifies that the service operates normally
// when Firebase is disabled (empty service_account_path).
func TestFirebaseDisabled_NoError(t *testing.T) {
	fbClient, err := pkgfirebase.New(config.FirebaseConfig{})
	if err != nil {
		t.Fatalf("New with empty path: unexpected error: %v", err)
	}
	if fbClient.IsEnabled() {
		t.Error("want IsEnabled()=false for empty path")
	}

	// All operations must be no-ops without error.
	ctx := context.Background()
	if err := fbClient.WriteOrderStatus(ctx, "u1", "o1", map[string]any{"status": "PENDING"}); err != nil {
		t.Errorf("WriteOrderStatus no-op: %v", err)
	}
	if err := fbClient.SendPush(ctx, []string{"token1"}, "title", "body", nil); err != nil {
		t.Errorf("SendPush no-op: %v", err)
	}
	fbClient.Close() // must not panic
}

// TestUnreadCount_HTTPEndpoint verifies the unread-count endpoint.
func TestUnreadCount_HTTPEndpoint(t *testing.T) {
	env := setupEnv(t)

	w := env.do(t, "GET", "/api/v1/me/notifications/unread-count", "")
	if w.Code != http.StatusOK {
		t.Fatalf("unread-count: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Count int64 `json:"Count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Data.Count != 0 {
		t.Errorf("initial unread count: want 0, got %d", resp.Data.Count)
	}
}
