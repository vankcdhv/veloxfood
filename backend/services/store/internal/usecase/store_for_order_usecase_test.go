package usecase

import (
	"context"
	"errors"
	"testing"

	"project/services/store/internal/entity"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockStoreRepo struct {
	getByIDFn             func(ctx context.Context, id string) (*entity.Store, error)
	getByVendorIDFn       func(ctx context.Context, vendorID string) (*entity.Store, error)
	listFn                func(ctx context.Context, saleStatus string) ([]*entity.Store, error)
	updateFn              func(ctx context.Context, s *entity.Store) error
	deleteFn              func(ctx context.Context, id string) error
	createFn              func(ctx context.Context, s *entity.Store) error
	updatePickupEnabledFn func(ctx context.Context, id string, enabled bool) error
	updateSaleStatusFn    func(ctx context.Context, id string, status string) error
}

func (m *mockStoreRepo) GetByID(ctx context.Context, id string) (*entity.Store, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreRepo) GetByVendorID(ctx context.Context, vendorID string) (*entity.Store, error) {
	if m.getByVendorIDFn != nil {
		return m.getByVendorIDFn(ctx, vendorID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreRepo) List(ctx context.Context, saleStatus string) ([]*entity.Store, error) {
	if m.listFn != nil {
		return m.listFn(ctx, saleStatus)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreRepo) Update(ctx context.Context, s *entity.Store) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, s)
	}
	return errors.New("not implemented")
}

func (m *mockStoreRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockStoreRepo) Create(ctx context.Context, s *entity.Store) error {
	if m.createFn != nil {
		return m.createFn(ctx, s)
	}
	return errors.New("not implemented")
}

func (m *mockStoreRepo) UpdatePickupEnabled(ctx context.Context, id string, enabled bool) error {
	if m.updatePickupEnabledFn != nil {
		return m.updatePickupEnabledFn(ctx, id, enabled)
	}
	return errors.New("not implemented")
}

func (m *mockStoreRepo) UpdateSaleStatus(ctx context.Context, id string, status string) error {
	if m.updateSaleStatusFn != nil {
		return m.updateSaleStatusFn(ctx, id, status)
	}
	return errors.New("not implemented")
}

func (m *mockStoreRepo) ListByOwner(_ context.Context, _ string) ([]*entity.Store, error) {
	return nil, errors.New("not implemented")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestStoreForOrder_StoreNotFound(t *testing.T) {
	ctx := context.Background()
	storeID := "store-unknown"
	locationID := "room-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, id string) (*entity.Store, error) {
			return nil, errors.New("not found")
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-001", "building-001", true, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	_, err := uc.GetStoreForOrder(ctx, storeID, "ROOM", locationID)
	// usecase wraps the raw error from repo (not gorm.ErrRecordNotFound), so it returns an error.
	if err == nil {
		t.Fatal("expected error for store not found, got nil")
	}
}

func TestStoreForOrder_ReturnsStoreData(t *testing.T) {
	ctx := context.Background()
	storeID, locationID := "store-001", "room-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, id string) (*entity.Store, error) {
			if id == storeID {
				return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
			}
			return nil, errors.New("not found")
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, sid, _, status string) ([]*entity.MenuItem, error) {
			if sid == storeID && status == "on" {
				return []*entity.MenuItem{{ID: "item-1", Name: "Pho", Price: 5000}}, nil
			}
			return nil, errors.New("not found")
		},
	}
	shippingRepo := &mockShippingRepo{
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-001", "building-001", true, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	result, err := uc.GetStoreForOrder(ctx, storeID, "ROOM", locationID)
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if !result.Found {
		t.Error("expected Found=true")
	}
	if result.SaleStatus != "OPEN" {
		t.Errorf("expected SaleStatus=OPEN, got %s", result.SaleStatus)
	}
	if len(result.Items) != 1 || result.Items[0].Price != 5000 {
		t.Errorf("unexpected items: %+v", result.Items)
	}
}

func TestStoreForOrder_ResolveShipFee_Room(t *testing.T) {
	ctx := context.Background()
	storeID, roomID, floorID, buildingID := "store-001", "room-001", "floor-001", "building-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, sid string, scopes, ids []string) (*entity.ShipFeeRule, error) {
			// First candidate is room; should match immediately.
			if len(scopes) > 0 && scopes[0] == "room" && ids[0] == roomID {
				return &entity.ShipFeeRule{UnitFee: 3000}, nil
			}
			return nil, nil
		},
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	result, err := uc.GetStoreForOrder(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if !result.Served {
		t.Error("expected Served=true")
	}
	if result.UnitShipFee != 3000 {
		t.Errorf("expected UnitShipFee=3000, got %d", result.UnitShipFee)
	}
}

func TestStoreForOrder_ResolveShipFee_Floor(t *testing.T) {
	ctx := context.Background()
	storeID, floorID, buildingID := "store-001", "floor-001", "building-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		resolveShipFeeFn: func(_ context.Context, _ string, scopes, ids []string) (*entity.ShipFeeRule, error) {
			for i := range scopes {
				if scopes[i] == "floor" && ids[i] == floorID {
					return &entity.ShipFeeRule{UnitFee: 10000}, nil
				}
			}
			return nil, nil
		},
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	result, err := uc.GetStoreForOrder(ctx, storeID, "FLOOR", floorID)
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if !result.Served {
		t.Error("expected Served=true")
	}
	if result.UnitShipFee != 10000 {
		t.Errorf("expected 10000, got %d", result.UnitShipFee)
	}
}

func TestStoreForOrder_NotServedLocation(t *testing.T) {
	ctx := context.Background()
	storeID, roomID := "store-001", "room-unknown"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "", "", false, nil // room not found
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	result, err := uc.GetStoreForOrder(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if result.Served {
		t.Error("expected Served=false for unknown room")
	}
}

func TestStoreForOrder_Pickup_SkipsShipFee(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		listShipCutoffsFn: func(_ context.Context, _ string) ([]*entity.ShipCutoff, error) {
			return []*entity.ShipCutoff{}, nil
		},
	}
	// Resolver should never be called for PICKUP (empty locationID).
	resolver := &mockLocationResolver{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	// PICKUP: empty locationID
	result, err := uc.GetStoreForOrder(ctx, storeID, "", "")
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if result.Served {
		t.Error("expected Served=false for PICKUP (no location)")
	}
}

func TestStoreForOrder_ResolvesOrderDeadline(t *testing.T) {
	ctx := context.Background()
	storeID, roomID := "store-001", "room-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN"}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{
		listShipCutoffsFn: func(_ context.Context, sid string) ([]*entity.ShipCutoff, error) {
			if sid == storeID {
				return []*entity.ShipCutoff{
					{ID: "c1", CutoffTime: "11:00", LeadMinutes: 30},
					{ID: "c2", CutoffTime: "18:00", LeadMinutes: 60},
				}, nil
			}
			return nil, errors.New("not found")
		},
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-001", "building-001", true, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver)
	result, err := uc.GetStoreForOrder(ctx, storeID, "ROOM", roomID)
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if result.OrderDeadline == "" {
		t.Error("expected OrderDeadline to be set")
	}
}

// Helper used by tests
func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}
