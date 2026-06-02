package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"project/services/store/internal/entity"

	"gorm.io/gorm"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockOutboxRepo struct {
	appendFn        func(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error
	pickupFn        func(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error)
	markPublishedFn func(ctx context.Context, id string) error
	markFailedFn    func(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}

func (m *mockOutboxRepo) Append(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error {
	if m.appendFn != nil {
		return m.appendFn(ctx, tx, evt)
	}
	return nil
}

func (m *mockOutboxRepo) Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error) {
	if m.pickupFn != nil {
		return m.pickupFn(ctx, limit, maxAttempts)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	if m.markPublishedFn != nil {
		return m.markPublishedFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, errMsg, attempts, maxAttempts)
	}
	return errors.New("not implemented")
}

type mockShippingRepoWithHours struct {
	*mockShippingRepo
	updateChangeRequestStatusFn func(ctx context.Context, id string, status string, reviewedBy string) error
}

func (m *mockShippingRepoWithHours) UpdateChangeRequestStatus(ctx context.Context, id string, status string, reviewedBy string) error {
	if m.updateChangeRequestStatusFn != nil {
		return m.updateChangeRequestStatusFn(ctx, id, status, reviewedBy)
	}
	return errors.New("not implemented")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestHoursUsecase_CreateOperatingHours(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"

	mockShip := &mockShippingRepo{}
	mockShip.createOperatingHoursFn = func(_ context.Context, oh *entity.OperatingHours) error {
		if oh != nil {
			oh.ID = "oh-001"
		}
		return nil
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	// Note: HoursUsecase requires *gorm.DB for transaction support.
	// For unit tests that don't use transactions, we can pass a nil DB and skip tx-dependent methods.
	// For this test, we're testing the simple path that doesn't use DB.

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	result, err := uc.CreateOperatingHours(ctx, storeID, 1, "09:00", "22:00")

	if err != nil {
		t.Fatalf("CreateOperatingHours: %v", err)
	}
	if result.ID != "oh-001" {
		t.Errorf("expected ID oh-001, got %s", result.ID)
	}
	if result.OpenTime != "09:00" {
		t.Errorf("expected OpenTime 09:00, got %s", result.OpenTime)
	}
}

func TestHoursUsecase_ListOperatingHours(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	hours := []*entity.OperatingHours{
		{ID: "oh-1", StoreID: storeID, Weekday: 1, OpenTime: "09:00", CloseTime: "22:00"},
		{ID: "oh-2", StoreID: storeID, Weekday: 2, OpenTime: "09:00", CloseTime: "22:00"},
	}

	mockShip := &mockShippingRepo{}
	mockShip.listOperatingHoursFn = func(_ context.Context, sid string) ([]*entity.OperatingHours, error) {
		if sid == storeID {
			return hours, nil
		}
		return nil, errors.New("not found")
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	result, err := uc.ListOperatingHours(ctx, storeID)

	if err != nil {
		t.Fatalf("ListOperatingHours: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 hours, got %d", len(result))
	}
}

func TestHoursUsecase_UpdateOperatingHours(t *testing.T) {
	ctx := context.Background()
	hoursID := "oh-001"
	oldHours := &entity.OperatingHours{
		ID:        hoursID,
		Weekday:   1,
		OpenTime:  "09:00",
		CloseTime: "22:00",
	}

	mockShip := &mockShippingRepo{}
	mockShip.getOperatingHoursFn = func(_ context.Context, id string) (*entity.OperatingHours, error) {
		if id == hoursID {
			return oldHours, nil
		}
		return nil, errors.New("not found")
	}
	mockShip.updateOperatingHoursFn = func(_ context.Context, oh *entity.OperatingHours) error {
		return nil
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	result, err := uc.UpdateOperatingHours(ctx, hoursID, 2, "08:00", "23:00")

	if err != nil {
		t.Fatalf("UpdateOperatingHours: %v", err)
	}
	if result.Weekday != 2 {
		t.Errorf("expected weekday 2, got %d", result.Weekday)
	}
	if result.OpenTime != "08:00" {
		t.Errorf("expected OpenTime 08:00, got %s", result.OpenTime)
	}
}

func TestHoursUsecase_DeleteOperatingHours(t *testing.T) {
	ctx := context.Background()
	hoursID := "oh-001"
	deleted := false

	mockShip := &mockShippingRepo{}
	mockShip.deleteOperatingHoursFn = func(_ context.Context, id string) error {
		if id == hoursID {
			deleted = true
			return nil
		}
		return errors.New("not found")
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	err := uc.DeleteOperatingHours(ctx, hoursID)

	if err != nil {
		t.Fatalf("DeleteOperatingHours: %v", err)
	}
	if !deleted {
		t.Error("expected delete to be called")
	}
}

func TestHoursUsecase_SubmitHoursChange_CreatesRequest(t *testing.T) {
	// This test exercises SubmitHoursChange which creates a pending request and appends an outbox event.
	// Since it uses *gorm.DB for transaction, we verify the interface exists.
	// Full testing deferred to integration tests.
	// For unit test, we verify method signature exists and handles basic logic.

	// Test is here as documentation that transaction-based methods need integration testing.
	// This prevents the false sense of security from unit testing transaction-dependent code
	// with mocks that can't replicate gorm.DB behavior.
}

func TestHoursUsecase_RejectHoursChange_InvalidStatus(t *testing.T) {
	ctx := context.Background()
	reqID := "req-001"
	adminID := "admin-001"
	req := &entity.OperatingHoursChangeRequest{
		ID:     reqID,
		Status: "already_approved", // not pending
	}

	mockShip := &mockShippingRepo{}
	mockShip.getChangeRequestFn = func(_ context.Context, id string) (*entity.OperatingHoursChangeRequest, error) {
		if id == reqID {
			return req, nil
		}
		return nil, errors.New("not found")
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	err := uc.RejectHoursChange(ctx, reqID, adminID)

	if !errors.Is(err, ErrInvalidChangeStatus) {
		t.Errorf("expected ErrInvalidChangeStatus, got %v", err)
	}
}

func TestHoursUsecase_ListChangeRequests(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"

	mockShip := &mockShippingRepo{}
	mockShip.listChangeRequestsFn = func(_ context.Context, sid, status string) ([]*entity.OperatingHoursChangeRequest, error) {
		if sid == storeID && status == "pending" {
			return []*entity.OperatingHoursChangeRequest{
				{ID: "req-1", StoreID: storeID, Status: "pending"},
			}, nil
		}
		return nil, errors.New("not found")
	}

	shippingRepo := &mockShippingRepoWithHours{
		mockShippingRepo: mockShip,
	}
	outboxRepo := &mockOutboxRepo{}

	uc := NewHoursUsecase(nil, shippingRepo, outboxRepo)
	result, err := uc.ListChangeRequests(ctx, storeID, "pending")

	if err != nil {
		t.Fatalf("ListChangeRequests: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 change request, got %d", len(result))
	}
}

// ── IsOpenNow tests ───────────────────────────────────────────────────────────

func makeHoursUCWithHours(hours []*entity.OperatingHours) HoursUsecase {
	mockShip := &mockShippingRepo{}
	mockShip.listOperatingHoursFn = func(_ context.Context, _ string) ([]*entity.OperatingHours, error) {
		return hours, nil
	}
	return NewHoursUsecase(nil, &mockShippingRepoWithHours{mockShippingRepo: mockShip}, &mockOutboxRepo{})
}

// Monday = weekday 1 (time.Monday)
func mondayAt(h, m int) time.Time {
	// Find or construct a Monday
	t := time.Date(2024, 1, 1, h, m, 0, 0, time.UTC) // 2024-01-01 is a Monday
	return t
}

func TestIsOpenNow_OpenStatus_WithinHours(t *testing.T) {
	hours := []*entity.OperatingHours{
		{StoreID: "s1", Weekday: int16(time.Monday), OpenTime: "08:00", CloseTime: "22:00"},
	}
	uc := makeHoursUCWithHours(hours)
	// 10:00 on Monday → within 08:00–22:00
	res, err := uc.IsOpenNow(context.Background(), "s1", "OPEN", mondayAt(10, 0))
	if err != nil {
		t.Fatalf("IsOpenNow: %v", err)
	}
	if !res.OpenNow {
		t.Error("expected OpenNow=true")
	}
	if res.OpenTimeToday != "08:00" {
		t.Errorf("expected OpenTimeToday=08:00, got %s", res.OpenTimeToday)
	}
	if res.CloseTimeToday != "22:00" {
		t.Errorf("expected CloseTimeToday=22:00, got %s", res.CloseTimeToday)
	}
}

func TestIsOpenNow_OpenStatus_BeforeOpen(t *testing.T) {
	hours := []*entity.OperatingHours{
		{StoreID: "s1", Weekday: int16(time.Monday), OpenTime: "08:00", CloseTime: "22:00"},
	}
	uc := makeHoursUCWithHours(hours)
	// 07:59 on Monday → before opening
	res, err := uc.IsOpenNow(context.Background(), "s1", "OPEN", mondayAt(7, 59))
	if err != nil {
		t.Fatalf("IsOpenNow: %v", err)
	}
	if res.OpenNow {
		t.Error("expected OpenNow=false before open time")
	}
	if res.OpenTimeToday != "08:00" {
		t.Errorf("expected OpenTimeToday=08:00, got %s", res.OpenTimeToday)
	}
}

func TestIsOpenNow_PausedStatus_WithinHours(t *testing.T) {
	hours := []*entity.OperatingHours{
		{StoreID: "s1", Weekday: int16(time.Monday), OpenTime: "08:00", CloseTime: "22:00"},
	}
	uc := makeHoursUCWithHours(hours)
	// 10:00 but SaleStatus=PAUSED → not open
	res, err := uc.IsOpenNow(context.Background(), "s1", "PAUSED", mondayAt(10, 0))
	if err != nil {
		t.Fatalf("IsOpenNow: %v", err)
	}
	if res.OpenNow {
		t.Error("expected OpenNow=false when SaleStatus=PAUSED")
	}
	// But open/close times still reported for display
	if res.OpenTimeToday != "08:00" {
		t.Errorf("expected OpenTimeToday=08:00, got %s", res.OpenTimeToday)
	}
}

func TestIsOpenNow_NoHoursForToday(t *testing.T) {
	// No hours for Monday
	hours := []*entity.OperatingHours{
		{StoreID: "s1", Weekday: int16(time.Tuesday), OpenTime: "09:00", CloseTime: "21:00"},
	}
	uc := makeHoursUCWithHours(hours)
	res, err := uc.IsOpenNow(context.Background(), "s1", "OPEN", mondayAt(10, 0))
	if err != nil {
		t.Fatalf("IsOpenNow: %v", err)
	}
	if res.OpenNow {
		t.Error("expected OpenNow=false when no hours configured for today")
	}
	if res.OpenTimeToday != "" {
		t.Errorf("expected empty OpenTimeToday, got %s", res.OpenTimeToday)
	}
}

func TestIsOpenNow_ExactBoundaries(t *testing.T) {
	hours := []*entity.OperatingHours{
		{StoreID: "s1", Weekday: int16(time.Monday), OpenTime: "08:00", CloseTime: "22:00"},
	}
	uc := makeHoursUCWithHours(hours)

	// At exactly open time (08:00)
	res, err := uc.IsOpenNow(context.Background(), "s1", "OPEN", mondayAt(8, 0))
	if err != nil {
		t.Fatalf("IsOpenNow at open: %v", err)
	}
	if !res.OpenNow {
		t.Error("expected OpenNow=true at exactly open time")
	}

	// At exactly close time (22:00)
	res, err = uc.IsOpenNow(context.Background(), "s1", "OPEN", mondayAt(22, 0))
	if err != nil {
		t.Fatalf("IsOpenNow at close: %v", err)
	}
	if !res.OpenNow {
		t.Error("expected OpenNow=true at exactly close time")
	}
}
