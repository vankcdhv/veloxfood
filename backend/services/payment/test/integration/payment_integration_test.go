// Package integration covers the payment service end-to-end against a running
// Postgres (payment_db_test). Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
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
	"project/pkg/outbox"
	"project/pkg/testutil"
	payevent "project/services/payment/internal/handler/event"
	handlerhttp "project/services/payment/internal/handler/http"
	v1 "project/services/payment/internal/handler/http/v1"
	"project/services/payment/internal/infrastructure/momo"
	"project/services/payment/internal/infrastructure/persistence"
	"project/services/payment/internal/repository"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ── path helpers ──────────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/payment.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// ── tables truncated between tests ───────────────────────────────────────────

var allTables = []string{
	"ledger_entries", "payments", "payout_batches",
	"wallets", "processed_events", "outbox_events",
}

// ── test environment ──────────────────────────────────────────────────────────

type testEnv struct {
	engine     *gin.Engine
	db         *gorm.DB
	cfg        *config.Config
	token      string
	customerID string
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

	customerID := "11111111-1111-1111-1111-111111111111"

	// Repos
	walletRepo := persistence.NewWalletGormRepository(db)
	ledgerRepo := persistence.NewLedgerGormRepository(db)
	paymentRepo := persistence.NewPaymentGormRepository(db)
	payoutRepo := persistence.NewPayoutGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)
	momoClient := momo.NewClient(cfg.MoMo)

	// Usecases
	walletUC := usecase.NewWalletUsecase(walletRepo, ledgerRepo)
	refundUC := usecase.NewRefundUsecase(db, walletRepo, ledgerRepo, paymentRepo, outboxRepo)
	topupUC := usecase.NewTopupUsecase(db, paymentRepo, momoClient, cfg.MoMo.IpnURL, cfg.MoMo.RedirectURL)
	ipnUC := usecase.NewIPNUsecase(db, walletRepo, ledgerRepo, paymentRepo, outboxRepo, momoClient)
	payoutUC := usecase.NewPayoutUsecase(db, walletRepo, ledgerRepo, payoutRepo, outboxRepo)
	_ = refundUC // used in individual test functions directly

	// HTTP engine with pass-through auth (user ID injected via middleware)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(pkgmiddleware.RequestMetadata())
	engine.Use(func(c *gin.Context) {
		ctx := authmw.WithUserID(c.Request.Context(), customerID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	handlerhttp.RegisterRoutes(engine, handlerhttp.RouterConfig{
		MoMoHandler:    v1.NewMoMoHandler(ipnUC),
		WalletHandler:  v1.NewWalletHandler(walletUC, topupUC),
		AdminHandler:   v1.NewAdminPayoutHandler(payoutUC),
		AuthMiddleware: func(c *gin.Context) { c.Next() },
	})

	// Mint real JWT where possible (falls back to pass-through token on Redis failure).
	token := "test-token"
	authStore, redisErr := authstore.NewRedisAuthStore(cfg.Redis)
	if redisErr == nil {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwtSvc := authjwt.NewRS256Service(priv, &priv.PublicKey, 15*time.Minute, 168*time.Hour)
		if pair, issueErr := jwtSvc.Issue(context.Background(), customerID); issueErr == nil {
			_ = authStore.Whitelist(context.Background(), pair.JTIAccess, customerID, pair.AccessTTL)
			token = pair.Access
		}
	}

	return &testEnv{engine: engine, db: db, cfg: cfg, token: token, customerID: customerID}
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

// ── Kafka envelope builder for synthetic event tests ─────────────────────────

// buildKafkaMsg wraps data in an outbox.Envelope and returns a kafka.Message.
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

// ── MoMo IPN signature helper ─────────────────────────────────────────────────

// signedIPNBody builds a correctly HMAC-SHA256-signed MoMo IPN body.
func signedIPNBody(t *testing.T, cfg config.MoMoConfig, orderID string, amount int64, transID int64, resultCode int) string {
	t.Helper()
	extraData := ""
	message := "Successful."
	orderInfo := "VeloxFood wallet topup"
	orderType := "momo_wallet"
	partnerCode := cfg.PartnerCode
	accessKey := cfg.AccessKey
	payType := "qr"
	requestID := "req-" + orderID
	responseTime := time.Now().UnixMilli()

	raw := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%d&resultCode=%d&transId=%d",
		accessKey, amount, extraData, message, orderID, orderInfo, orderType,
		partnerCode, payType, requestID, responseTime, resultCode, transID,
	)
	mac := hmac.New(sha256.New, []byte(cfg.SecretKey))
	mac.Write([]byte(raw))
	sig := hex.EncodeToString(mac.Sum(nil))

	body := map[string]any{
		"partnerCode":  partnerCode,
		"accessKey":    accessKey,
		"requestId":    requestID,
		"amount":       amount,
		"orderId":      orderID,
		"orderInfo":    orderInfo,
		"orderType":    orderType,
		"transId":      transID,
		"resultCode":   resultCode,
		"message":      message,
		"payType":      payType,
		"responseTime": responseTime,
		"extraData":    extraData,
		"signature":    sig,
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// ── delivery handler factory for direct invocation in tests ──────────────────

type kafkaHandlerFunc func(ctx context.Context, msg kafka.Message) error

func newDeliveryHandlerForTest(
	db *gorm.DB,
	paymentRepo repository.PaymentRepository,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	outboxRepo repository.OutboxRepository,
	processedRepo repository.ProcessedEventRepository,
) kafkaHandlerFunc {
	h := payevent.NewDeliveryEventHandler(db, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo)
	return h.HandleKafkaMessage
}

func newOrderHandlerForTest(
	db *gorm.DB,
	paymentRepo repository.PaymentRepository,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	outboxRepo repository.OutboxRepository,
	processedRepo repository.ProcessedEventRepository,
	refundUC usecase.RefundUsecase,
) kafkaHandlerFunc {
	h := payevent.NewOrderEventHandler(db, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo, refundUC)
	return h.HandleKafkaMessage
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestGetWallet_ZeroBalance(t *testing.T) {
	env := setupEnv(t)
	w := env.do(t, "GET", "/api/v1/me/wallet", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Balance int64 `json:"Balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Data.Balance != 0 {
		t.Errorf("expected balance=0, got %d", resp.Data.Balance)
	}
}

// TestTopup_SignedIPN_CreditsWallet seeds a PENDING topup payment, fires a
// correctly signed IPN, and verifies balance + ledger + idempotency.
func TestTopup_SignedIPN_CreditsWallet(t *testing.T) {
	env := setupEnv(t)

	// Seed SYSTEM wallet — topup IPN debits SYSTEM (gateway collected cash) and credits CUSTOMER.
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	topupID := "aaaaaaaa-0001-0001-0001-000000000001"
	iKey := "topup:" + topupID
	env.db.Exec(
		"INSERT INTO payments (id, topup_id, customer_id, method, provider, env, amount, status, idempotency_key) VALUES (gen_random_uuid(), $1, $2, 'MOMO', 'MOMO', 'demo', $3, 'PENDING', $4)",
		topupID, env.customerID, int64(50000), iKey,
	)

	body := signedIPNBody(t, env.cfg.MoMo, topupID, 50000, 12345, 0)
	w := env.do(t, "POST", "/api/v1/payments/momo/ipn", body)
	if w.Code != http.StatusOK {
		t.Fatalf("IPN: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var balance int64
	env.db.Raw("SELECT w.balance FROM wallets w WHERE w.owner_type='CUSTOMER' AND w.owner_id=$1", env.customerID).Scan(&balance)
	if balance != 50000 {
		t.Errorf("expected balance=50000 after topup IPN, got %d", balance)
	}

	var ledgerCount int64
	env.db.Raw("SELECT COUNT(*) FROM ledger_entries WHERE ref_id=$1 AND entry_type='TOPUP' AND amount>0", topupID).Scan(&ledgerCount)
	if ledgerCount != 1 {
		t.Errorf("expected 1 TOPUP credit ledger entry, got %d", ledgerCount)
	}

	// Idempotency: second IPN must not double-credit.
	w2 := env.do(t, "POST", "/api/v1/payments/momo/ipn", body)
	if w2.Code != http.StatusOK {
		t.Fatalf("second IPN: expected 200, got %d", w2.Code)
	}
	env.db.Raw("SELECT w.balance FROM wallets w WHERE w.owner_type='CUSTOMER' AND w.owner_id=$1", env.customerID).Scan(&balance)
	if balance != 50000 {
		t.Errorf("idempotent IPN: balance should still be 50000, got %d", balance)
	}
}

func TestTopup_InvalidSignature_Rejected(t *testing.T) {
	env := setupEnv(t)
	body := `{"partnerCode":"MOMO","accessKey":"bad","requestId":"r1","amount":10000,"orderId":"fake","orderInfo":"x","orderType":"momo_wallet","transId":1,"resultCode":0,"message":"ok","payType":"qr","responseTime":0,"extraData":"","signature":"invalidsig"}`
	w := env.do(t, "POST", "/api/v1/payments/momo/ipn", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid signature: expected 400, got %d", w.Code)
	}
}

func TestCapture_Wallet_DebitsAndCredits(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'CUSTOMER',$1,100000)", env.customerID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	orderID := "bbbbbbbb-0001-0001-0001-000000000001"
	result, err := captureUC.Capture(context.Background(), usecase.CaptureRequest{
		OrderID:    orderID,
		CustomerID: env.customerID,
		Amount:     30000,
		Method:     "WALLET",
	})
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	if result.Status != "CAPTURED" {
		t.Errorf("expected CAPTURED, got %s (err: %s)", result.Status, result.Error)
	}

	var custBalance int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='CUSTOMER' AND owner_id=$1", env.customerID).Scan(&custBalance)
	if custBalance != 70000 {
		t.Errorf("customer balance: expected 70000, got %d", custBalance)
	}

	// Double-entry: 2 entries, sum = 0.
	var entries []struct{ Amount int64 }
	env.db.Raw("SELECT amount FROM ledger_entries WHERE ref_id=$1", orderID).Scan(&entries)
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(entries))
	}
	var sum int64
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 0 {
		t.Errorf("double-entry invariant: sum=%d (want 0)", sum)
	}

	// balance_after on debit entry must match actual wallet balance.
	var debitBalanceAfter int64
	env.db.Raw("SELECT balance_after FROM ledger_entries WHERE ref_id=$1 AND amount<0", orderID).Scan(&debitBalanceAfter)
	if debitBalanceAfter != custBalance {
		t.Errorf("balance_after mismatch: ledger=%d wallet=%d", debitBalanceAfter, custBalance)
	}
}

func TestCapture_Wallet_Idempotent(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'CUSTOMER',$1,100000)", env.customerID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	orderID := "cccccccc-0001-0001-0001-000000000001"
	req := usecase.CaptureRequest{OrderID: orderID, CustomerID: env.customerID, Amount: 20000, Method: "WALLET"}

	r1, _ := captureUC.Capture(context.Background(), req)
	r2, _ := captureUC.Capture(context.Background(), req)

	if r1.PaymentID != r2.PaymentID {
		t.Errorf("idempotent Capture: different payment IDs %s vs %s", r1.PaymentID, r2.PaymentID)
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM ledger_entries WHERE ref_id=$1", orderID).Scan(&count)
	if count != 2 {
		t.Errorf("idempotent capture: expected 2 ledger entries, got %d", count)
	}
}

func TestCapture_MoMo_ReturnsPendingPaymentRow(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)

	orderID := "dddddddd-0001-0001-0001-000000000001"
	result, err := captureUC.Capture(context.Background(), usecase.CaptureRequest{
		OrderID:    orderID,
		CustomerID: env.customerID,
		Amount:     50000,
		Method:     "MOMO",
	})
	if err != nil {
		t.Fatalf("Capture(MOMO): unexpected error: %v", err)
	}
	if result.PaymentID == "" {
		t.Error("expected non-empty payment_id")
	}

	// Payment row must exist (PENDING or FAILED — network availability varies in CI).
	var payStatus string
	env.db.Raw("SELECT status FROM payments WHERE order_id=$1", orderID).Scan(&payStatus)
	if payStatus == "" {
		t.Error("expected payment row in DB after Capture(MOMO)")
	}
}

func TestRefund_CreditsCustomerWallet(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)
	refundUC := usecase.NewRefundUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo)

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'CUSTOMER',$1,100000)", env.customerID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	orderID := "eeeeeeee-0001-0001-0001-000000000001"
	_, err := captureUC.Capture(context.Background(), usecase.CaptureRequest{
		OrderID:    orderID,
		CustomerID: env.customerID,
		Amount:     40000,
		Method:     "WALLET",
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	var balAfterCapture int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='CUSTOMER' AND owner_id=$1", env.customerID).Scan(&balAfterCapture)

	_, err = refundUC.Refund(context.Background(), orderID, 40000)
	if err != nil {
		t.Fatalf("refund: %v", err)
	}

	var balAfterRefund int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='CUSTOMER' AND owner_id=$1", env.customerID).Scan(&balAfterRefund)
	if balAfterRefund != balAfterCapture+40000 {
		t.Errorf("expected balance %d after refund, got %d", balAfterCapture+40000, balAfterRefund)
	}

	var status string
	env.db.Raw("SELECT status FROM payments WHERE order_id=$1", orderID).Scan(&status)
	if status != "REFUNDED" {
		t.Errorf("expected payment REFUNDED, got %s", status)
	}
}

func TestRefund_Idempotent(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)
	refundUC := usecase.NewRefundUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo)

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'CUSTOMER',$1,100000)", env.customerID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	orderID := "ffffffff-0001-0001-0001-000000000001"
	captureUC.Capture(context.Background(), usecase.CaptureRequest{OrderID: orderID, CustomerID: env.customerID, Amount: 20000, Method: "WALLET"})

	r1, err1 := refundUC.Refund(context.Background(), orderID, 20000)
	r2, err2 := refundUC.Refund(context.Background(), orderID, 20000)
	if err1 != nil || err2 != nil {
		t.Fatalf("refund errors: %v %v", err1, err2)
	}
	if r1.RefundID != r2.RefundID {
		t.Errorf("idempotent refund returned different IDs")
	}

	// Balance must be restored exactly once (not doubled).
	var bal int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='CUSTOMER' AND owner_id=$1", env.customerID).Scan(&bal)
	if bal != 100000 {
		t.Errorf("expected balance=100000 after idempotent refund, got %d", bal)
	}
}

func TestProcessedEvents_Deduplication(t *testing.T) {
	env := setupEnv(t)

	processedRepo := persistence.NewProcessedEventGormRepository(env.db)
	eventID := "99999999-0001-0001-0001-000000000001"

	inserted1, err := processedRepo.MarkProcessed(context.Background(), env.db, eventID)
	if err != nil {
		t.Fatalf("first MarkProcessed: %v", err)
	}
	if !inserted1 {
		t.Error("expected inserted=true on first call")
	}

	inserted2, err := processedRepo.MarkProcessed(context.Background(), env.db, eventID)
	if err != nil {
		t.Fatalf("second MarkProcessed: %v", err)
	}
	if inserted2 {
		t.Error("expected inserted=false on duplicate event_id")
	}
}

// TestOrderCancelled_SyntheticEnvelope feeds a synthetic order.cancelled
// envelope directly to the handler and verifies the customer wallet is refunded.
func TestOrderCancelled_SyntheticEnvelope_RefundsCustomer(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	processedRepo := persistence.NewProcessedEventGormRepository(env.db)
	captureUC := usecase.NewCaptureUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
		momo.NewClient(env.cfg.MoMo), env.cfg.MoMo.IpnURL, env.cfg.MoMo.RedirectURL)
	refundUC := usecase.NewRefundUsecase(env.db, walletRepo, ledgerRepo, paymentRepo, outboxRepo)

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'CUSTOMER',$1,100000)", env.customerID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	orderID := "aaaabbbb-0001-0001-0001-000000000001"
	captureUC.Capture(context.Background(), usecase.CaptureRequest{
		OrderID:    orderID,
		CustomerID: env.customerID,
		Amount:     25000,
		Method:     "WALLET",
	})

	handler := newOrderHandlerForTest(env.db, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo, refundUC)
	data, _ := json.Marshal(map[string]any{
		"order_id":        orderID,
		"was_paid_online": true,
		"amount":          int64(25000),
	})
	msg := buildKafkaMsg("order.cancelled", data)
	if err := handler(context.Background(), msg); err != nil {
		t.Fatalf("order.cancelled handler: %v", err)
	}

	var bal int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='CUSTOMER' AND owner_id=$1", env.customerID).Scan(&bal)
	if bal != 100000 {
		t.Errorf("expected wallet restored to 100000, got %d", bal)
	}
}

// TestOrderDelivered_COD_CreditsStorePayable feeds a synthetic order.delivered
// envelope for a COD order and verifies STORE_PAYABLE is credited.
func TestOrderDelivered_COD_CreditsStorePayable(t *testing.T) {
	env := setupEnv(t)

	paymentRepo := persistence.NewPaymentGormRepository(env.db)
	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	processedRepo := persistence.NewProcessedEventGormRepository(env.db)

	storeID := "55555555-5555-5555-5555-555555555555"
	orderID := "66666666-6666-6666-6666-666666666666"

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")
	env.db.Exec(
		"INSERT INTO payments (id,order_id,customer_id,method,provider,env,amount,status) VALUES (gen_random_uuid(),$1,$2,'COD','INTERNAL','demo',35000,'PENDING')",
		orderID, env.customerID,
	)

	handler := newDeliveryHandlerForTest(env.db, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo)

	data, _ := json.Marshal(map[string]any{"order_id": orderID, "store_id": storeID, "amount": int64(35000)})
	msg := buildKafkaMsg("order.delivered", data)
	if err := handler(context.Background(), msg); err != nil {
		t.Fatalf("order.delivered handler: %v", err)
	}

	var storeBal int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='STORE_PAYABLE' AND owner_id=$1", storeID).Scan(&storeBal)
	if storeBal != 35000 {
		t.Errorf("STORE_PAYABLE balance: expected 35000, got %d", storeBal)
	}

	var status string
	env.db.Raw("SELECT status FROM payments WHERE order_id=$1", orderID).Scan(&status)
	if status != "CAPTURED" {
		t.Errorf("expected payment CAPTURED, got %s", status)
	}

	// Idempotency: replay the same message — balance must not double.
	if err := handler(context.Background(), buildKafkaMsg("order.delivered", data)); err != nil {
		t.Errorf("idempotent order.delivered: unexpected error: %v", err)
	}
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='STORE_PAYABLE' AND owner_id=$1", storeID).Scan(&storeBal)
	if storeBal != 35000 {
		t.Errorf("idempotent: STORE_PAYABLE should remain 35000, got %d", storeBal)
	}
}

func TestPayout_ExecuteAtomic_IdempotentSkipSettled(t *testing.T) {
	env := setupEnv(t)

	walletRepo := persistence.NewWalletGormRepository(env.db)
	ledgerRepo := persistence.NewLedgerGormRepository(env.db)
	payoutRepo := persistence.NewPayoutGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	payoutUC := usecase.NewPayoutUsecase(env.db, walletRepo, ledgerRepo, payoutRepo, outboxRepo)

	storeID := "77777777-7777-7777-7777-777777777777"
	adminID := "88888888-8888-8888-8888-888888888888"

	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'STORE_PAYABLE',$1,60000)", storeID)
	env.db.Exec("INSERT INTO wallets (id,owner_type,owner_id,balance) VALUES (gen_random_uuid(),'SYSTEM','00000000-0000-0000-0000-000000000000',999999999) ON CONFLICT DO NOTHING")

	batch, err := payoutUC.CreateBatch(context.Background(), usecase.CreatePayoutRequest{
		StoreID:     storeID,
		TotalAmount: 60000,
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	if err := payoutUC.ExecuteBatch(context.Background(), batch.ID, adminID); err != nil {
		t.Fatalf("ExecuteBatch: %v", err)
	}

	var storeBal int64
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='STORE_PAYABLE' AND owner_id=$1", storeID).Scan(&storeBal)
	if storeBal != 0 {
		t.Errorf("STORE_PAYABLE balance: expected 0 after payout, got %d", storeBal)
	}

	var status string
	env.db.Raw("SELECT status FROM payout_batches WHERE id=$1", batch.ID).Scan(&status)
	if status != "SETTLED" {
		t.Errorf("expected SETTLED, got %s", status)
	}

	// Execute again — must be idempotent (no error, balance stays 0).
	if err := payoutUC.ExecuteBatch(context.Background(), batch.ID, adminID); err != nil {
		t.Errorf("idempotent ExecuteBatch: unexpected error: %v", err)
	}
	env.db.Raw("SELECT balance FROM wallets WHERE owner_type='STORE_PAYABLE' AND owner_id=$1", storeID).Scan(&storeBal)
	if storeBal != 0 {
		t.Errorf("idempotent: STORE_PAYABLE balance should remain 0, got %d", storeBal)
	}
}
