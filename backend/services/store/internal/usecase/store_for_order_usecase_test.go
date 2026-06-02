package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"project/services/store/internal/entity"
)

// ── mockStoreRepo ─────────────────────────────────────────────────────────────

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

// ── mockCatalogRepo (minimal — only methods used by StoreForOrderUsecase) ──────

type mockCatalogRepo struct {
	listMenuItemsFn func(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error)
}

func (m *mockCatalogRepo) CreateCategory(ctx context.Context, c *entity.Category) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) GetCategory(ctx context.Context, id string) (*entity.Category, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) UpdateCategory(ctx context.Context, c *entity.Category) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DeleteCategory(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) CreateMenuItem(ctx context.Context, item *entity.MenuItem) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) ListMenuItems(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error) {
	if m.listMenuItemsFn != nil {
		return m.listMenuItemsFn(ctx, storeID, categoryID, status)
	}
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) UpdateMenuItem(ctx context.Context, item *entity.MenuItem) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) ToggleMenuItemStatus(ctx context.Context, id string, status string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DeleteMenuItem(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) CreateOptionGroup(ctx context.Context, og *entity.OptionGroup) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) GetOptionGroup(ctx context.Context, id string) (*entity.OptionGroup, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) UpdateOptionGroup(ctx context.Context, og *entity.OptionGroup) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DeleteOptionGroup(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) CreateOption(ctx context.Context, o *entity.Option) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) GetOption(ctx context.Context, id string) (*entity.Option, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) UpdateOption(ctx context.Context, o *entity.Option) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DeleteOption(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) AttachOptionGroup(ctx context.Context, mio *entity.MenuItemOption) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DetachOptionGroup(ctx context.Context, menuItemID string, optionGroupID string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) CreateCombo(ctx context.Context, c *entity.Combo) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) GetCombo(ctx context.Context, id string) (*entity.Combo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockCatalogRepo) UpdateCombo(ctx context.Context, c *entity.Combo) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) DeleteCombo(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) AddComboItem(ctx context.Context, ci *entity.ComboItem) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) RemoveComboItem(ctx context.Context, comboID string, menuItemID string) error {
	return errors.New("not implemented")
}
func (m *mockCatalogRepo) ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error) {
	return nil, errors.New("not implemented")
}

// ── mockHoursUsecase ──────────────────────────────────────────────────────────

type mockHoursUsecase struct {
	isOpenNowFn func(ctx context.Context, storeID string, storeStatus string, t time.Time) (OpenNowResult, error)
}

func (m *mockHoursUsecase) CreateOperatingHours(ctx context.Context, storeID string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHoursUsecase) ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHoursUsecase) UpdateOperatingHours(ctx context.Context, id string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHoursUsecase) DeleteOperatingHours(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *mockHoursUsecase) IsOpenNow(ctx context.Context, storeID string, storeStatus string, t time.Time) (OpenNowResult, error) {
	if m.isOpenNowFn != nil {
		return m.isOpenNowFn(ctx, storeID, storeStatus, t)
	}
	return OpenNowResult{}, nil
}
func (m *mockHoursUsecase) SubmitHoursChange(ctx context.Context, storeID string, payload any) (*entity.OperatingHoursChangeRequest, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHoursUsecase) ApproveHoursChange(ctx context.Context, reqID, adminID string) error {
	return errors.New("not implemented")
}
func (m *mockHoursUsecase) RejectHoursChange(ctx context.Context, reqID, adminID string) error {
	return errors.New("not implemented")
}
func (m *mockHoursUsecase) ListChangeRequests(ctx context.Context, storeID, status string) ([]*entity.OperatingHoursChangeRequest, error) {
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
	shippingRepo := &mockShippingRepo{}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-001", "building-001", true, nil
		},
	}
	hoursUC := &mockHoursUsecase{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
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
				return &entity.Store{ID: storeID, SaleStatus: "OPEN", PrepMinutes: 20}, nil
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
	shippingRepo := &mockShippingRepo{}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "floor-001", "building-001", true, nil
		},
	}
	hoursUC := &mockHoursUsecase{
		isOpenNowFn: func(_ context.Context, _ string, _ string, _ time.Time) (OpenNowResult, error) {
			return OpenNowResult{OpenNow: true, OpenTimeToday: "08:00", CloseTimeToday: "22:00"}, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
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
	if result.PrepMinutes != 20 {
		t.Errorf("expected PrepMinutes=20, got %d", result.PrepMinutes)
	}
	if !result.OpenNow {
		t.Error("expected OpenNow=true")
	}
	if result.OpenTimeToday != "08:00" {
		t.Errorf("expected OpenTimeToday=08:00, got %s", result.OpenTimeToday)
	}
	if result.CloseTimeToday != "22:00" {
		t.Errorf("expected CloseTimeToday=22:00, got %s", result.CloseTimeToday)
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
	}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return floorID, buildingID, true, nil
		},
	}
	hoursUC := &mockHoursUsecase{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
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
	}
	resolver := &mockLocationResolver{
		getFloorFn: func(_ context.Context, _ string) (string, bool, error) {
			return buildingID, true, nil
		},
	}
	hoursUC := &mockHoursUsecase{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
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
	shippingRepo := &mockShippingRepo{}
	resolver := &mockLocationResolver{
		getRoomFn: func(_ context.Context, _ string) (string, string, bool, error) {
			return "", "", false, nil // room not found
		},
	}
	hoursUC := &mockHoursUsecase{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
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
	shippingRepo := &mockShippingRepo{}
	// Resolver should never be called for PICKUP (empty locationID).
	resolver := &mockLocationResolver{}
	hoursUC := &mockHoursUsecase{}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
	// PICKUP: empty locationID
	result, err := uc.GetStoreForOrder(ctx, storeID, "", "")
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if result.Served {
		t.Error("expected Served=false for PICKUP (no location)")
	}
}

func TestStoreForOrder_OpenNow_PopulatedFromHoursUC(t *testing.T) {
	ctx := context.Background()
	storeID := "store-001"

	storeRepo := &mockStoreRepo{
		getByIDFn: func(_ context.Context, _ string) (*entity.Store, error) {
			return &entity.Store{ID: storeID, SaleStatus: "OPEN", PrepMinutes: 30}, nil
		},
	}
	catalogRepo := &mockCatalogRepo{
		listMenuItemsFn: func(_ context.Context, _, _, _ string) ([]*entity.MenuItem, error) {
			return []*entity.MenuItem{}, nil
		},
	}
	shippingRepo := &mockShippingRepo{}
	resolver := &mockLocationResolver{}
	hoursUC := &mockHoursUsecase{
		isOpenNowFn: func(_ context.Context, sid, status string, _ time.Time) (OpenNowResult, error) {
			if sid == storeID && status == "OPEN" {
				return OpenNowResult{OpenNow: true, OpenTimeToday: "09:00", CloseTimeToday: "21:00"}, nil
			}
			return OpenNowResult{}, nil
		},
	}

	uc := NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, resolver, hoursUC)
	result, err := uc.GetStoreForOrder(ctx, storeID, "", "")
	if err != nil {
		t.Fatalf("GetStoreForOrder: %v", err)
	}
	if !result.OpenNow {
		t.Error("expected OpenNow=true")
	}
	if result.OpenTimeToday != "09:00" {
		t.Errorf("expected OpenTimeToday=09:00, got %s", result.OpenTimeToday)
	}
	if result.CloseTimeToday != "21:00" {
		t.Errorf("expected CloseTimeToday=21:00, got %s", result.CloseTimeToday)
	}
	if result.PrepMinutes != 30 {
		t.Errorf("expected PrepMinutes=30, got %d", result.PrepMinutes)
	}
}
