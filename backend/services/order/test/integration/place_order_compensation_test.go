// Compensation-path tests for the REAL place-order saga (not the simplified
// wrapper): each downstream failure must roll back what came before it, and a
// failing rollback call must leave a durable pending_compensations row.
package integration

import (
	"context"
	"errors"
	"sync"
	"testing"

	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/infrastructure/persistence"
	"project/services/order/internal/usecase"
)

// recordingPromo counts saga calls and can be scripted to fail.
type recordingPromo struct {
	mu          sync.Mutex
	confirmErr  error
	releaseErr  error
	applyCalls  int
	confirms    int
	releases    int
}

func (p *recordingPromo) ApplyPromotion(_ context.Context, _, _, _ string, _ []string, _ int64, _ int32) (*grpcclient.PromotionApplyResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.applyCalls++
	return &grpcclient.PromotionApplyResult{Success: true, ItemDiscount: 5000}, nil
}

func (p *recordingPromo) QuotePromotion(_ context.Context, _, _ string, _ []string, _ int64, _ int32) (*grpcclient.PromotionApplyResult, error) {
	return &grpcclient.PromotionApplyResult{Success: true, ItemDiscount: 5000}, nil
}

func (p *recordingPromo) ConfirmUsage(_ context.Context, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.confirms++
	return p.confirmErr
}

func (p *recordingPromo) ReleaseUsage(_ context.Context, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.releases++
	return p.releaseErr
}

// recordingPayment can fail Capture and/or Refund.
type recordingPayment struct {
	mu         sync.Mutex
	captureErr error
	refundErr  error
	captures   int
	refunds    int
}

func (p *recordingPayment) Capture(_ context.Context, _, _ string, _ int64, _ string) (*grpcclient.PaymentCaptureResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.captures++
	if p.captureErr != nil {
		return nil, p.captureErr
	}
	return &grpcclient.PaymentCaptureResult{Status: "CAPTURED"}, nil
}

func (p *recordingPayment) Refund(_ context.Context, _ string, _ int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refunds++
	return p.refundErr
}

func (p *recordingPayment) GetPaymentStatus(_ context.Context, _ string) (*grpcclient.PaymentStatusResult, error) {
	return &grpcclient.PaymentStatusResult{Status: "PAID"}, nil
}

func placeReq() usecase.PlaceOrderRequest {
	return usecase.PlaceOrderRequest{
		CustomerID:    testCustomerID,
		StoreID:       testStoreID,
		Fulfillment:   entity.FulfillmentPickup,
		PaymentMethod: entity.MethodWallet,
		VoucherCodes:  []string{"SAVE5"},
		Items:         []usecase.PlaceOrderItem{{MenuItemID: testItemID, Qty: 1}},
	}
}

func newRealPlaceOrderUC(t *testing.T, env *testEnv, promo *recordingPromo, pay *recordingPayment) usecase.PlaceOrderUsecase {
	t.Helper()
	orderRepo := persistence.NewOrderGormRepository(env.db)
	cartRepo := persistence.NewCartGormRepository(env.db)
	outboxRepo := persistence.NewOutboxGormRepository(env.db)
	compRepo := persistence.NewCompensationGormRepository(env.db)
	// Engine "inline" — these tests exercise the hand-rolled saga; the DTM
	// path needs a live coordinator and is covered by the E2E smoke.
	return usecase.NewPlaceOrderUsecase(env.db, orderRepo, cartRepo, outboxRepo, compRepo, &stubStore{}, promo, pay,
		usecase.SagaSettings{Engine: "inline"})
}

func TestPlaceOrderSaga_CaptureFails_ReleasesPromotion(t *testing.T) {
	env := setupEnv(t)
	promo := &recordingPromo{}
	pay := &recordingPayment{captureErr: errors.New("wallet service down")}
	uc := newRealPlaceOrderUC(t, env, promo, pay)

	if _, err := uc.PlaceOrder(context.Background(), placeReq()); err == nil {
		t.Fatal("expected place order to fail when capture fails")
	}

	if promo.releases != 1 {
		t.Errorf("ReleaseUsage calls: want 1, got %d", promo.releases)
	}
	var orders int64
	env.db.Raw("SELECT count(*) FROM orders").Scan(&orders)
	if orders != 0 {
		t.Errorf("orders persisted: want 0, got %d", orders)
	}
}

func TestPlaceOrderSaga_ConfirmFails_RefundsAndReleases(t *testing.T) {
	env := setupEnv(t)
	promo := &recordingPromo{confirmErr: errors.New("promotion db down")}
	pay := &recordingPayment{}
	uc := newRealPlaceOrderUC(t, env, promo, pay)

	if _, err := uc.PlaceOrder(context.Background(), placeReq()); err == nil {
		t.Fatal("expected place order to fail when confirm fails")
	}

	if pay.refunds != 1 {
		t.Errorf("Refund calls: want 1, got %d", pay.refunds)
	}
	if promo.releases != 1 {
		t.Errorf("ReleaseUsage calls: want 1, got %d", promo.releases)
	}
}

func TestPlaceOrderSaga_CompensationFails_PersistsPendingRow(t *testing.T) {
	env := setupEnv(t)
	// Confirm fails → compensation kicks in, and BOTH rollback calls also fail:
	// each must leave a durable pending_compensations row for the worker.
	promo := &recordingPromo{
		confirmErr: errors.New("promotion db down"),
		releaseErr: errors.New("still down"),
	}
	pay := &recordingPayment{refundErr: errors.New("payment down too")}
	uc := newRealPlaceOrderUC(t, env, promo, pay)

	if _, err := uc.PlaceOrder(context.Background(), placeReq()); err == nil {
		t.Fatal("expected place order to fail")
	}

	var actions []string
	env.db.Raw("SELECT action FROM pending_compensations WHERE done_at IS NULL ORDER BY action").Scan(&actions)
	if len(actions) != 2 || actions[0] != "REFUND" || actions[1] != "RELEASE_USAGE" {
		t.Fatalf("pending_compensations: want [REFUND RELEASE_USAGE], got %v", actions)
	}
}
