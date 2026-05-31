// Package integration tests the delivery service end-to-end against a real
// Postgres (delivery_db_test). Set SKIP_INTEGRATION=1 to skip.
package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"project/pkg/config"
	"project/pkg/outbox"
	"project/pkg/testutil"
	deliveryevent "project/services/delivery/internal/handler/event"
	"project/services/delivery/internal/infrastructure/persistence"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/usecase"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ── path helpers ───────────────────────────────────────────────────────────────

func configPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/delivery.yaml"))
}

func migrationsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../migrations"))
}

// ── constants ──────────────────────────────────────────────────────────────────

const (
	testOrderID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testStoreID    = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	testLocationID = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	testCustomerID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	testShipperID  = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	testShipper2   = "11111111-1111-1111-1111-111111111111"
)

var allTables = []string{
	"delivery_incidents", "deliveries", "delivery_batches",
	"outbox_events", "processed_events",
}

// ── test environment ───────────────────────────────────────────────────────────

type testEnv struct {
	db           *gorm.DB
	deliveryUC   usecase.DeliveryUsecase
	incidentUC   usecase.IncidentUsecase
	orderHandler *deliveryevent.OrderEventHandler
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

	deliveryRepo := persistence.NewDeliveryGormRepository(db)
	batchRepo := persistence.NewBatchGormRepository(db)
	incidentRepo := persistence.NewIncidentGormRepository(db)
	outboxRepo := persistence.NewOutboxGormRepository(db)
	processedRepo := persistence.NewProcessedEventGormRepository(db)

	// nil locationClient — room path resolution gracefully returns empty string in tests
	deliveryUC := usecase.NewDeliveryUsecase(db, deliveryRepo, batchRepo, outboxRepo, nil)
	incidentUC := usecase.NewIncidentUsecase(db, deliveryRepo, incidentRepo, outboxRepo)
	orderHandler := deliveryevent.NewOrderEventHandler(db, deliveryRepo, processedRepo)

	return &testEnv{
		db:           db,
		deliveryUC:   deliveryUC,
		incidentUC:   incidentUC,
		orderHandler: orderHandler,
	}
}

// ── Kafka message builder ──────────────────────────────────────────────────────

func buildKafkaMsg(eventType string, data any) kafka.Message {
	raw, _ := json.Marshal(data)
	env := outbox.Envelope{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Data:       raw,
	}
	b, _ := json.Marshal(env)
	return kafka.Message{Topic: "order.events", Value: b}
}

func orderReadyMsg(orderID, fulfillment string) kafka.Message {
	return buildKafkaMsg("order.ready", map[string]any{
		"order_id":    orderID,
		"status":      "READY",
		"store_id":    testStoreID,
		"location_id": testLocationID,
		"ship_fee":    int64(15000),
		"fulfillment": fulfillment,
		"customer_id": testCustomerID,
	})
}

// ── Tests: order.ready consumer ────────────────────────────────────────────────

// TestOrderReady_Delivery_CreatesAvailable verifies a DELIVERY order.ready
// event creates an AVAILABLE delivery record.
func TestOrderReady_Delivery_CreatesAvailable(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	if err := env.orderHandler.HandleKafkaMessage(ctx, orderReadyMsg(testOrderID, "DELIVERY")); err != nil {
		t.Fatalf("handle order.ready: %v", err)
	}

	var status string
	env.db.Raw("SELECT status FROM deliveries WHERE order_id = ?", testOrderID).Scan(&status)
	if status != "AVAILABLE" {
		t.Errorf("want status=AVAILABLE, got %q", status)
	}

	var shipFee int64
	env.db.Raw("SELECT ship_fee FROM deliveries WHERE order_id = ?", testOrderID).Scan(&shipFee)
	if shipFee != 15000 {
		t.Errorf("want ship_fee=15000, got %d", shipFee)
	}
}

// TestOrderReady_Pickup_SkipsDelivery verifies PICKUP orders produce no delivery record.
func TestOrderReady_Pickup_SkipsDelivery(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	if err := env.orderHandler.HandleKafkaMessage(ctx, orderReadyMsg(testOrderID, "PICKUP")); err != nil {
		t.Fatalf("handle order.ready PICKUP: %v", err)
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM deliveries WHERE order_id = ?", testOrderID).Scan(&count)
	if count != 0 {
		t.Errorf("PICKUP: want 0 delivery records, got %d", count)
	}
}

// TestOrderReady_Idempotent verifies the same event_id cannot create two records.
func TestOrderReady_Idempotent(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	msg := orderReadyMsg(testOrderID, "DELIVERY")
	for i := 0; i < 2; i++ {
		if err := env.orderHandler.HandleKafkaMessage(ctx, msg); err != nil {
			t.Fatalf("handle iteration %d: %v", i, err)
		}
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM deliveries WHERE order_id = ?", testOrderID).Scan(&count)
	if count != 1 {
		t.Errorf("idempotent: want 1 delivery record, got %d", count)
	}
}

// ── Tests: atomic claim ────────────────────────────────────────────────────────

// TestClaim_HappyPath verifies a shipper can claim an AVAILABLE delivery.
func TestClaim_HappyPath(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)

	result, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, testShipperID)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if result.BatchID == "" {
		t.Error("batch_id must be non-empty")
	}

	var status, shipperID string
	env.db.Raw("SELECT status, shipper_id FROM deliveries WHERE order_id = ?", testOrderID).
		Row().Scan(&status, &shipperID)
	if status != "CLAIMED" {
		t.Errorf("want CLAIMED, got %q", status)
	}
	if shipperID != testShipperID {
		t.Errorf("want shipper_id=%s, got %s", testShipperID, shipperID)
	}

	var outboxCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='delivery.claimed'").Scan(&outboxCount)
	if outboxCount != 1 {
		t.Errorf("want 1 delivery.claimed outbox row, got %d", outboxCount)
	}
}

// TestClaim_ConcurrentRace verifies exactly one of two concurrent claims wins.
func TestClaim_ConcurrentRace(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)

	type result struct{ err error }
	ch := make(chan result, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for _, shipperID := range []string{testShipperID, testShipper2} {
		shipperID := shipperID
		go func() {
			defer wg.Done()
			_, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, shipperID)
			ch <- result{err}
		}()
	}
	wg.Wait()
	close(ch)

	successes := 0
	for r := range ch {
		if r.err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("concurrent claim: want exactly 1 success, got %d", successes)
	}
}

// TestClaim_BatchMax5_Enforced verifies a shipper cannot exceed 5 active deliveries
// per (shipper, store) batch.
func TestClaim_BatchMax5_Enforced(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	orderIDs := make([]string, 6)
	for i := range orderIDs {
		orderIDs[i] = uuid.NewString()
		seedAvailableDelivery(t, env.db, orderIDs[i])
	}

	for i := 0; i < 5; i++ {
		if _, err := env.deliveryUC.ClaimDelivery(ctx, orderIDs[i], testShipperID); err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
	}

	_, err := env.deliveryUC.ClaimDelivery(ctx, orderIDs[5], testShipperID)
	if err != usecase.ErrBatchFull {
		t.Errorf("6th claim: want ErrBatchFull, got %v", err)
	}
}

// ── Tests: status transitions ──────────────────────────────────────────────────

// TestStatusTransitions_FullChain exercises CLAIMED→PICKED_UP→DELIVERING→DELIVERED.
func TestStatusTransitions_FullChain(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)
	if _, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, testShipperID); err != nil {
		t.Fatalf("claim: %v", err)
	}

	steps := []entity.DeliveryStatus{
		entity.DeliveryPickedUp,
		entity.DeliveryDelivering,
		entity.DeliveryDelivered,
	}
	for _, s := range steps {
		if err := env.deliveryUC.UpdateStatus(ctx, testOrderID, testShipperID, s); err != nil {
			t.Fatalf("update to %s: %v", s, err)
		}
	}

	var status string
	var deliveredAt *time.Time
	env.db.Raw("SELECT status, delivered_at FROM deliveries WHERE order_id = ?", testOrderID).
		Row().Scan(&status, &deliveredAt)

	if status != "DELIVERED" {
		t.Errorf("want DELIVERED, got %q", status)
	}
	if deliveredAt == nil {
		t.Error("delivered_at must be set on DELIVERED")
	}

	var scCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='delivery.status_changed'").Scan(&scCount)
	if scCount != 3 {
		t.Errorf("want 3 delivery.status_changed rows, got %d", scCount)
	}
}

// TestStatusTransition_InvalidTransition verifies skipping PICKED_UP is rejected.
func TestStatusTransition_InvalidTransition(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)
	if _, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, testShipperID); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Jump CLAIMED→DELIVERING (skipping PICKED_UP) — must be rejected.
	err := env.deliveryUC.UpdateStatus(ctx, testOrderID, testShipperID, entity.DeliveryDelivering)
	if err != usecase.ErrInvalidTransition {
		t.Errorf("want ErrInvalidTransition, got %v", err)
	}
}

// TestStatusTransition_WrongShipper verifies another shipper cannot update.
func TestStatusTransition_WrongShipper(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)
	if _, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, testShipperID); err != nil {
		t.Fatalf("claim: %v", err)
	}

	err := env.deliveryUC.UpdateStatus(ctx, testOrderID, testShipper2, entity.DeliveryPickedUp)
	if err != usecase.ErrNotDeliveryOwner {
		t.Errorf("wrong shipper: want ErrNotDeliveryOwner, got %v", err)
	}
}

// ── Tests: cutoff_reached and cancelled ───────────────────────────────────────

// TestCutoffReached_SetsStoreDelivering verifies order.cutoff_reached moves
// the AVAILABLE delivery to STORE_DELIVERING.
func TestCutoffReached_SetsStoreDelivering(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	if err := env.orderHandler.HandleKafkaMessage(ctx, orderReadyMsg(testOrderID, "DELIVERY")); err != nil {
		t.Fatalf("order.ready: %v", err)
	}

	msg := buildKafkaMsg("order.cutoff_reached", map[string]any{
		"order_id": testOrderID,
		"store_id": testStoreID,
	})
	if err := env.orderHandler.HandleKafkaMessage(ctx, msg); err != nil {
		t.Fatalf("order.cutoff_reached: %v", err)
	}

	var status string
	env.db.Raw("SELECT status FROM deliveries WHERE order_id = ?", testOrderID).Scan(&status)
	if status != "STORE_DELIVERING" {
		t.Errorf("want STORE_DELIVERING, got %q", status)
	}
}

// TestOrderCancelled_CancelsDelivery verifies order.cancelled sets delivery CANCELLED.
func TestOrderCancelled_CancelsDelivery(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	if err := env.orderHandler.HandleKafkaMessage(ctx, orderReadyMsg(testOrderID, "DELIVERY")); err != nil {
		t.Fatalf("order.ready: %v", err)
	}

	msg := buildKafkaMsg("order.cancelled", map[string]any{
		"order_id": testOrderID,
	})
	if err := env.orderHandler.HandleKafkaMessage(ctx, msg); err != nil {
		t.Fatalf("order.cancelled: %v", err)
	}

	var status string
	env.db.Raw("SELECT status FROM deliveries WHERE order_id = ?", testOrderID).Scan(&status)
	if status != "CANCELLED" {
		t.Errorf("want CANCELLED, got %q", status)
	}
}

// ── Tests: incident ────────────────────────────────────────────────────────────

// TestReportIncident_CreatesIncidentAndOutboxRow verifies incident creation + event.
func TestReportIncident_CreatesIncidentAndOutboxRow(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	seedAvailableDelivery(t, env.db, testOrderID)
	if _, err := env.deliveryUC.ClaimDelivery(ctx, testOrderID, testShipperID); err != nil {
		t.Fatalf("claim: %v", err)
	}

	inc, err := env.incidentUC.ReportIncident(ctx, usecase.ReportIncidentRequest{
		OrderID:   testOrderID,
		ShipperID: testShipperID,
		Type:      "WRONG_ADDRESS",
		Note:      "Customer not at location",
	})
	if err != nil {
		t.Fatalf("report incident: %v", err)
	}
	if inc.ID == "" {
		t.Error("incident ID must be set")
	}

	var outboxCount int64
	env.db.Raw("SELECT COUNT(*) FROM outbox_events WHERE event_type='delivery.incident_reported'").Scan(&outboxCount)
	if outboxCount != 1 {
		t.Errorf("want 1 delivery.incident_reported outbox row, got %d", outboxCount)
	}
}

// ── Tests: processed_events deduplication ─────────────────────────────────────

// TestProcessedEvents_Dedup verifies the same event_id is only processed once.
func TestProcessedEvents_Dedup(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	msg := orderReadyMsg(testOrderID, "DELIVERY")
	for i := 0; i < 2; i++ {
		if err := env.orderHandler.HandleKafkaMessage(ctx, msg); err != nil {
			t.Fatalf("handle iteration %d: %v", i, err)
		}
	}

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM deliveries WHERE order_id = ?", testOrderID).Scan(&count)
	if count != 1 {
		t.Errorf("dedup: want 1 delivery, got %d", count)
	}

	var processedCount int64
	env.db.Raw("SELECT COUNT(*) FROM processed_events").Scan(&processedCount)
	if processedCount != 1 {
		t.Errorf("dedup: want 1 processed_events row, got %d", processedCount)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

func seedAvailableDelivery(t *testing.T, db *gorm.DB, orderID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO deliveries (order_id, store_id, location_id, customer_id, ship_fee, status) VALUES (?,?,?,?,?,?)",
		orderID, testStoreID, testLocationID, testCustomerID, int64(15000), "AVAILABLE",
	).Error; err != nil {
		t.Fatalf("seed delivery %s: %v", orderID, err)
	}
}
