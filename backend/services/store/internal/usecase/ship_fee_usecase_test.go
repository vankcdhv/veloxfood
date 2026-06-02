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
	resolveShipFeeFn       func(ctx context.Context, storeID string, scopes, ids []string) (*entity.ShipFeeRule, error)
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

func (m *mockShippingRepo) ResolveShipFee(ctx context.Context, storeID string, scopes, ids []string) (*entity.ShipFeeRule, error) {
	if m.resolveShipFeeFn != nil {
		return m.resolveShipFeeFn(ctx, storeID, scopes, ids)
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

func (m *mockShippingRepo) ListShipCutoffsForStores(ctx context.Context, storeIDs []string) (map[string][]*entity.ShipCutoff, error) {
	return map[string][]*entity.ShipCutoff{}, nil
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

// mockLocationResolver implements usecase.LocationResolver.
type mockLocationResolver struct {
	getRoomFn     func(ctx context.Context, roomID string) (floorID, buildingID string, found bool, err error)
	getFloorFn    func(ctx context.Context, floorID string) (buildingID string, found bool, err error)
	getBuildingFn func(ctx context.Context, buildingID string) (found bool, err error)
}

func (m *mockLocationResolver) GetRoom(ctx context.Context, roomID string) (floorID, buildingID string, found bool, err error) {
	if m.getRoomFn != nil {
		return m.getRoomFn(ctx, roomID)
	}
	return "", "", false, errors.New("not implemented")
}

func (m *mockLocationResolver) GetFloor(ctx context.Context, floorID string) (buildingID string, found bool, err error) {
	if m.getFloorFn != nil {
		return m.getFloorFn(ctx, floorID)
	}
	return "", false, errors.New("not implemented")
}

func (m *mockLocationResolver) GetBuilding(ctx context.Context, buildingID string) (found bool, err error) {
	if m.getBuildingFn != nil {
		return m.getBuildingFn(ctx, buildingID)
	}
	return false, errors.New("not implemented")
}

// ── Helper: cascade priority tracker ─────────────────────────────────────────

// firstMatchResolver simulates the repo returning the first candidate that
// matches one of the provided (scope,id) pairs — mirrors persistence logic.
func firstMatchRule(rules map[string]*entity.ShipFeeRule) func(ctx context.Context, storeID string, scopes, ids []string) (*entity.ShipFeeRule, error) {
	return func(_ context.Context, _ string, scopes, ids []string) (*entity.ShipFeeRule, error) {
		for i := range scopes {
			key := scopes[i] + ":" + ids[i]
			if r, ok := rules[key]; ok {
				return r, nil
			}
		}
		return nil, nil
	}
}

// ── Tests: ResolveFee (3-level cascade) ───────────────────────────────────────

func TestResolveFee_Room_RoomRuleWins(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID, floorID, buildingID := "room-1", "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"room:room-1":         {Scope: "room", RefID: roomID, UnitFee: 5000},
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 5000 {
		t.Errorf("expected room fee 5000, got %d", fee)
	}
}

func TestResolveFee_Room_FloorRuleFallback(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID, floorID, buildingID := "room-1", "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"floor:floor-1":       {Scope: "floor", RefID: floorID, UnitFee: 10000},
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 10000 {
		t.Errorf("expected floor fee 10000, got %d", fee)
	}
}

func TestResolveFee_Room_BuildingRuleFallback(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID, floorID, buildingID := "room-1", "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 15000 {
		t.Errorf("expected building fee 15000, got %d", fee)
	}
}

func TestResolveFee_Floor_FloorRuleWins(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	floorID, buildingID := "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"floor:floor-1":       {Scope: "floor", RefID: floorID, UnitFee: 10000},
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "FLOOR", floorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 10000 {
		t.Errorf("expected floor fee 10000, got %d", fee)
	}
}

func TestResolveFee_Floor_BuildingRuleFallback(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	floorID, buildingID := "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "FLOOR", floorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 15000 {
		t.Errorf("expected building fee 15000, got %d", fee)
	}
}

func TestResolveFee_Building_DirectRule(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	buildingID := "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getBuildingFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "BUILDING", buildingID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 15000 {
		t.Errorf("expected 15000, got %d", fee)
	}
}

func TestResolveFee_ZeroFeeIsValid(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"
	roomID, floorID, buildingID := "room-vip", "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"room:room-vip": {Scope: "room", RefID: roomID, UnitFee: 0},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	fee, err := uc.ResolveFee(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 0 {
		t.Errorf("expected fee 0 (free), got %d", fee)
	}
}

func TestResolveFee_NotServed_NoRule(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, _ string, _, _ []string) (*entity.ShipFeeRule, error) {
			return nil, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-1", "building-1", true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.ResolveFee(ctx, "store-001", "ROOM", "room-1")
	if !errors.Is(err, ErrLocationNotServed) {
		t.Errorf("expected ErrLocationNotServed, got %v", err)
	}
}

func TestResolveFee_NotServed_RoomNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "", "", false, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.ResolveFee(ctx, "store-001", "ROOM", "room-unknown")
	if !errors.Is(err, ErrLocationNotServed) {
		t.Errorf("expected ErrLocationNotServed, got %v", err)
	}
}

func TestResolveFee_DefaultLevelIsRoom(t *testing.T) {
	ctx := context.Background()
	roomID, floorID, buildingID := "room-1", "floor-1", "building-1"

	rules := map[string]*entity.ShipFeeRule{
		"building:building-1": {Scope: "building", RefID: buildingID, UnitFee: 15000},
	}
	repo := &mockShippingRepo{resolveShipFeeFn: firstMatchRule(rules)}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	// empty level → defaults to ROOM
	fee, err := uc.ResolveFee(ctx, "store-001", "", roomID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fee != 15000 {
		t.Errorf("expected 15000, got %d", fee)
	}
}

// ── Tests: CreateShipFeeRule ───────────────────────────────────────────────────

func TestCreateShipFeeRule_Building_NoAncestorCheck(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{
		createShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error {
			r.ID = "rule-new"
			return nil
		},
	}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	rule, err := uc.CreateShipFeeRule(ctx, "store-001", "building", "building-001", 15000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.ID == "" {
		t.Error("expected ID to be assigned")
	}
}

func TestCreateShipFeeRule_DuplicateReturnsConflict(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{
		createShipFeeRuleFn: func(_ context.Context, _ *entity.ShipFeeRule) error {
			// Simulate the Postgres unique-violation on (store_id, scope, ref_id).
			return errors.New(`ERROR: duplicate key value violates unique constraint "uq_ship_fee_rule" (SQLSTATE 23505)`)
		},
	}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.CreateShipFeeRule(ctx, "store-001", "building", "building-001", 15000)
	if !errors.Is(err, ErrShipFeeRuleExists) {
		t.Errorf("expected ErrShipFeeRuleExists, got %v", err)
	}
}

func TestCreateShipFeeRule_Floor_RequiresBuildingRule(t *testing.T) {
	ctx := context.Background()
	floorID, buildingID := "floor-1", "building-1"

	repo := &mockShippingRepo{
		listShipFeeRulesFn: func(_ context.Context, _ string) ([]*entity.ShipFeeRule, error) {
			// No building rule yet.
			return []*entity.ShipFeeRule{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.CreateShipFeeRule(ctx, "store-001", "floor", floorID, 10000)
	if !errors.Is(err, ErrBuildingRuleRequired) {
		t.Errorf("expected ErrBuildingRuleRequired, got %v", err)
	}
}

func TestCreateShipFeeRule_Floor_SucceedsWhenBuildingRuleExists(t *testing.T) {
	ctx := context.Background()
	floorID, buildingID := "floor-1", "building-1"

	repo := &mockShippingRepo{
		listShipFeeRulesFn: func(_ context.Context, _ string) ([]*entity.ShipFeeRule, error) {
			return []*entity.ShipFeeRule{
				{Scope: "building", RefID: buildingID, UnitFee: 15000},
			}, nil
		},
		createShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error {
			r.ID = "rule-floor"
			return nil
		},
	}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	rule, err := uc.CreateShipFeeRule(ctx, "store-001", "floor", floorID, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Scope != "floor" {
		t.Errorf("expected scope floor, got %s", rule.Scope)
	}
}

func TestCreateShipFeeRule_Room_RequiresBuildingRule(t *testing.T) {
	ctx := context.Background()
	roomID, floorID, buildingID := "room-1", "floor-1", "building-1"

	repo := &mockShippingRepo{
		listShipFeeRulesFn: func(_ context.Context, _ string) ([]*entity.ShipFeeRule, error) {
			return []*entity.ShipFeeRule{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.CreateShipFeeRule(ctx, "store-001", "room", roomID, 5000)
	if !errors.Is(err, ErrBuildingRuleRequired) {
		t.Errorf("expected ErrBuildingRuleRequired, got %v", err)
	}
}

func TestCreateShipFeeRule_ZeroFeeAllowed(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{
		createShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error {
			r.ID = "rule-free"
			return nil
		},
	}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	rule, err := uc.CreateShipFeeRule(ctx, "store-001", "building", "building-001", 0)
	if err != nil {
		t.Fatalf("expected zero fee to be allowed, got: %v", err)
	}
	if rule.UnitFee != 0 {
		t.Errorf("expected fee 0, got %d", rule.UnitFee)
	}
}

func TestCreateShipFeeRule_NegativeFeeRejected(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.CreateShipFeeRule(ctx, "store-001", "building", "building-001", -1)
	if !errors.Is(err, ErrInvalidUnitFee) {
		t.Errorf("expected ErrInvalidUnitFee, got %v", err)
	}
}

func TestCreateShipFeeRule_InvalidScope(t *testing.T) {
	ctx := context.Background()
	repo := &mockShippingRepo{}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	_, err := uc.CreateShipFeeRule(ctx, "store-001", "country", "id-1", 1000)
	if !errors.Is(err, ErrInvalidScope) {
		t.Errorf("expected ErrInvalidScope, got %v", err)
	}
}

// ── Tests: UpdateShipFeeRule / DeleteShipFeeRule ───────────────────────────────

func TestUpdateShipFeeRule(t *testing.T) {
	ctx := context.Background()
	ruleID := "rule-001"
	oldRule := &entity.ShipFeeRule{ID: ruleID, StoreID: "store-001", UnitFee: 1000}

	repo := &mockShippingRepo{
		getShipFeeRuleFn: func(_ context.Context, id string) (*entity.ShipFeeRule, error) {
			if id == ruleID {
				return oldRule, nil
			}
			return nil, errors.New("not found")
		},
		updateShipFeeRuleFn: func(_ context.Context, r *entity.ShipFeeRule) error { return nil },
	}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	updated, err := uc.UpdateShipFeeRule(ctx, ruleID, 3000)
	if err != nil {
		t.Fatalf("UpdateShipFeeRule: %v", err)
	}
	if updated.UnitFee != 3000 {
		t.Errorf("expected fee 3000, got %d", updated.UnitFee)
	}
}

func TestDeleteShipFeeRule(t *testing.T) {
	ctx := context.Background()
	ruleID := "rule-001"
	deleted := false

	repo := &mockShippingRepo{
		deleteShipFeeRuleFn: func(_ context.Context, id string) error {
			if id == ruleID {
				deleted = true
				return nil
			}
			return errors.New("not found")
		},
	}
	resolver := &mockLocationResolver{}

	uc := NewShipFeeUsecase(repo, resolver)
	if err := uc.DeleteShipFeeRule(ctx, ruleID); err != nil {
		t.Fatalf("DeleteShipFeeRule: %v", err)
	}
	if !deleted {
		t.Error("expected delete to be called")
	}
}
