package usecase

import (
	"context"
	"errors"
	"testing"

	"project/services/store/internal/entity"

	"gorm.io/gorm"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockShippingRepo struct {
	createShipFeeRuleFn    func(ctx context.Context, r *entity.ShipFeeRule) error
	getShipFeeRuleFn       func(ctx context.Context, id string) (*entity.ShipFeeRule, error)
	listShipFeeRulesFn     func(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error)
	updateShipFeeRuleFn    func(ctx context.Context, r *entity.ShipFeeRule) error
	deleteShipFeeRuleFn    func(ctx context.Context, id string) error
	resolveShipFeeFn       func(ctx context.Context, storeID, buildingID, roomID string) (*entity.ShipFeeRule, error)
	createOperatingHoursFn func(ctx context.Context, oh *entity.OperatingHours) error
	getOperatingHoursFn    func(ctx context.Context, id string) (*entity.OperatingHours, error)
	listOperatingHoursFn   func(ctx context.Context, storeID string) ([]*entity.OperatingHours, error)
	updateOperatingHoursFn func(ctx context.Context, oh *entity.OperatingHours) error
	deleteOperatingHoursFn func(ctx context.Context, id string) error
	createShipCutoffFn     func(ctx context.Context, sc *entity.ShipCutoff) error
	getShipCutoffFn        func(ctx context.Context, id string) (*entity.ShipCutoff, error)
	listShipCutoffsFn      func(ctx context.Context, storeID string) ([]*entity.ShipCutoff, error)
	updateShipCutoffFn     func(ctx context.Context, sc *entity.ShipCutoff) error
	deleteShipCutoffFn     func(ctx context.Context, id string) error
	createChangeRequestFn  func(ctx context.Context, req *entity.OperatingHoursChangeRequest) error
	getChangeRequestFn     func(ctx context.Context, id string) (*entity.OperatingHoursChangeRequest, error)
	listChangeRequestsFn   func(ctx context.Context, storeID, status string) ([]*entity.OperatingHoursChangeRequest, error)
}

func (m *mockShippingRepo) CreateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error {
	if m.createShipFeeRuleFn != nil {
		return m.createShipFeeRuleFn(ctx, r)
	}
	return nil
}

func (m *mockShippingRepo) GetShipFeeRule(ctx context.Context, id string) (*entity.ShipFeeRule, error) {
	if m.getShipFeeRuleFn != nil {
		return m.getShipFeeRuleFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error) {
	if m.listShipFeeRulesFn != nil {
		return m.listShipFeeRulesFn(ctx, storeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) UpdateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error {
	if m.updateShipFeeRuleFn != nil {
		return m.updateShipFeeRuleFn(ctx, r)
	}
	return nil
}

func (m *mockShippingRepo) DeleteShipFeeRule(ctx context.Context, id string) error {
	if m.deleteShipFeeRuleFn != nil {
		return m.deleteShipFeeRuleFn(ctx, id)
	}
	return nil
}

func (m *mockShippingRepo) ResolveShipFee(ctx context.Context, storeID, buildingID, roomID string) (*entity.ShipFeeRule, error) {
	if m.resolveShipFeeFn != nil {
		return m.resolveShipFeeFn(ctx, storeID, buildingID, roomID)
	}
	return nil, errors.New("not implemented")
}

// stubs for other repository methods
func (m *mockShippingRepo) CreateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error {
	if m.createOperatingHoursFn != nil {
		return m.createOperatingHoursFn(ctx, oh)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) GetOperatingHours(ctx context.Context, id string) (*entity.OperatingHours, error) {
	if m.getOperatingHoursFn != nil {
		return m.getOperatingHoursFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error) {
	if m.listOperatingHoursFn != nil {
		return m.listOperatingHoursFn(ctx, storeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) UpdateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error {
	if m.updateOperatingHoursFn != nil {
		return m.updateOperatingHoursFn(ctx, oh)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) DeleteOperatingHours(ctx context.Context, id string) error {
	if m.deleteOperatingHoursFn != nil {
		return m.deleteOperatingHoursFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) CreateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error {
	if m.createShipCutoffFn != nil {
		return m.createShipCutoffFn(ctx, sc)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) GetShipCutoff(ctx context.Context, id string) (*entity.ShipCutoff, error) {
	if m.getShipCutoffFn != nil {
		return m.getShipCutoffFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) ListShipCutoffs(ctx context.Context, storeID string) ([]*entity.ShipCutoff, error) {
	if m.listShipCutoffsFn != nil {
		return m.listShipCutoffsFn(ctx, storeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) UpdateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error {
	if m.updateShipCutoffFn != nil {
		return m.updateShipCutoffFn(ctx, sc)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) DeleteShipCutoff(ctx context.Context, id string) error {
	if m.deleteShipCutoffFn != nil {
		return m.deleteShipCutoffFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) CreateChangeRequest(ctx context.Context, req *entity.OperatingHoursChangeRequest) error {
	if m.createChangeRequestFn != nil {
		return m.createChangeRequestFn(ctx, req)
	}
	return errors.New("not implemented")
}

func (m *mockShippingRepo) GetChangeRequest(ctx context.Context, id string) (*entity.OperatingHoursChangeRequest, error) {
	if m.getChangeRequestFn != nil {
		return m.getChangeRequestFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) ListChangeRequests(ctx context.Context, storeID string, status string) ([]*entity.OperatingHoursChangeRequest, error) {
	if m.listChangeRequestsFn != nil {
		return m.listChangeRequestsFn(ctx, storeID, status)
	}
	return nil, errors.New("not implemented")
}

func (m *mockShippingRepo) UpdateChangeRequestStatus(ctx context.Context, id string, status string, reviewedBy string) error {
	return errors.New("not implemented")
}

func (m *mockShippingRepo) DeleteOperatingHoursByStore(_ context.Context, _ *gorm.DB, _ string) error {
	return nil
}

func (m *mockShippingRepo) BulkCreateOperatingHours(_ context.Context, _ *gorm.DB, _ []*entity.OperatingHours) error {
	return nil
}

func (m *mockShippingRepo) DeleteShipCutoffsByStore(_ context.Context, _ *gorm.DB, _ string) error {
	return nil
}

func (m *mockShippingRepo) BulkCreateShipCutoffs(_ context.Context, _ *gorm.DB, _ []*entity.ShipCutoff) error {
	return nil
}

type mockRoomResolver struct {
	getRoomFn func(ctx context.Context, roomID string) (buildingID string, found bool, err error)
}

func (m *mockRoomResolver) GetRoom(ctx context.Context, roomID string) (string, bool, error) {
	if m.getRoomFn != nil {
		return m.getRoomFn(ctx, roomID)
	}
	return "", false, errors.New("not implemented")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestShipFeeUsecase_ResolveFee_RoomRuleBeatsBuilding(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID := "room-001"
	buildingID := "building-001"

	shippingRepo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, s, b, r string) (*entity.ShipFeeRule, error) {
			// Simulate room rule exists (preferred over building)
			if r == roomID {
				return &entity.ShipFeeRule{ID: "rule-room", Scope: "room", RefID: roomID, UnitFee: 5000}, nil
			}
			return nil, nil
		},
	}
	roomResolver := &mockRoomResolver{
		getRoomFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	fee, err := uc.ResolveFee(ctx, storeID, roomID)

	if err != nil {
		t.Fatalf("ResolveFee: %v", err)
	}
	if fee != 5000 {
		t.Errorf("expected fee 5000, got %d", fee)
	}
}

func TestShipFeeUsecase_ResolveFee_BuildingRuleWhenNoRoom(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID := "room-001"
	buildingID := "building-001"

	shippingRepo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, s, b, r string) (*entity.ShipFeeRule, error) {
			// Only building rule exists
			if b == buildingID {
				return &entity.ShipFeeRule{ID: "rule-building", Scope: "building", RefID: buildingID, UnitFee: 3000}, nil
			}
			return nil, nil
		},
	}
	roomResolver := &mockRoomResolver{
		getRoomFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	fee, err := uc.ResolveFee(ctx, storeID, roomID)

	if err != nil {
		t.Fatalf("ResolveFee: %v", err)
	}
	if fee != 3000 {
		t.Errorf("expected fee 3000, got %d", fee)
	}
}

func TestShipFeeUsecase_ResolveFee_NotServed(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID := "room-001"
	buildingID := "building-001"

	shippingRepo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, s, b, r string) (*entity.ShipFeeRule, error) {
			// No matching rule
			return nil, nil
		},
	}
	roomResolver := &mockRoomResolver{
		getRoomFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	_, err := uc.ResolveFee(ctx, storeID, roomID)

	if !errors.Is(err, ErrLocationNotServed) {
		t.Errorf("expected ErrLocationNotServed, got %v", err)
	}
}

func TestShipFeeUsecase_ResolveFee_RoomNotFound(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID := "room-unknown"

	shippingRepo := &mockShippingRepo{}
	roomResolver := &mockRoomResolver{
		getRoomFn: func(_ context.Context, _ string) (string, bool, error) {
			return "", false, nil // room not found
		},
	}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	_, err := uc.ResolveFee(ctx, storeID, roomID)

	if !errors.Is(err, ErrLocationNotServed) {
		t.Errorf("expected ErrLocationNotServed, got %v", err)
	}
}

func TestShipFeeUsecase_CreateShipFeeRule(t *testing.T) {
	ctx := context.Background()
	shippingRepo := &mockShippingRepo{
		createShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error {
			r.ID = "rule-new"
			return nil
		},
	}
	roomResolver := &mockRoomResolver{}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	rule, err := uc.CreateShipFeeRule(ctx, "store-001", "building", "building-001", 2000)

	if err != nil {
		t.Fatalf("CreateShipFeeRule: %v", err)
	}
	if rule.ID == "" {
		t.Error("expected rule ID to be assigned")
	}
	if rule.UnitFee != 2000 {
		t.Errorf("expected fee 2000, got %d", rule.UnitFee)
	}
}

func TestShipFeeUsecase_UpdateShipFeeRule(t *testing.T) {
	ctx := context.Background()
	ruleID := "rule-001"
	oldRule := &entity.ShipFeeRule{ID: ruleID, StoreID: "store-001", UnitFee: 1000}

	shippingRepo := &mockShippingRepo{
		getShipFeeRuleFn: func(_ context.Context, id string) (*entity.ShipFeeRule, error) {
			if id == ruleID {
				return oldRule, nil
			}
			return nil, errors.New("not found")
		},
		updateShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error {
			return nil
		},
	}
	roomResolver := &mockRoomResolver{}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	updated, err := uc.UpdateShipFeeRule(ctx, ruleID, 3000)

	if err != nil {
		t.Fatalf("UpdateShipFeeRule: %v", err)
	}
	if updated.UnitFee != 3000 {
		t.Errorf("expected fee 3000, got %d", updated.UnitFee)
	}
}

func TestShipFeeUsecase_DeleteShipFeeRule(t *testing.T) {
	ctx := context.Background()
	ruleID := "rule-001"
	deleted := false

	shippingRepo := &mockShippingRepo{
		deleteShipFeeRuleFn: func(_ context.Context, id string) error {
			if id == ruleID {
				deleted = true
				return nil
			}
			return errors.New("not found")
		},
	}
	roomResolver := &mockRoomResolver{}

	uc := NewShipFeeUsecase(shippingRepo, roomResolver)
	err := uc.DeleteShipFeeRule(ctx, ruleID)

	if err != nil {
		t.Fatalf("DeleteShipFeeRule: %v", err)
	}
	if !deleted {
		t.Error("expected delete to be called")
	}
}
