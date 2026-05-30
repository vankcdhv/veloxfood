package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockCatalogRepo struct {
	getSlotQuotaFn         func(ctx context.Context, menuItemID string, date time.Time, cutoffID string) (*entity.MenuItemSlotQuota, error)
	upsertSlotQuotaFn      func(ctx context.Context, q *entity.MenuItemSlotQuota) error
	decrementSlotQuotaFn   func(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error
	restoreSlotQuotaFn     func(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error
	listSlotQuotasFn       func(ctx context.Context, menuItemID string, date time.Time) ([]*entity.MenuItemSlotQuota, error)
	createCategoryFn       func(ctx context.Context, c *entity.Category) error
	getCategoryFn          func(ctx context.Context, id string) (*entity.Category, error)
	listCategoriesFn       func(ctx context.Context, storeID string) ([]*entity.Category, error)
	updateCategoryFn       func(ctx context.Context, c *entity.Category) error
	deleteCategoryFn       func(ctx context.Context, id string) error
	createMenuItemFn       func(ctx context.Context, m *entity.MenuItem) error
	getMenuItemFn          func(ctx context.Context, id string) (*entity.MenuItem, error)
	listMenuItemsFn        func(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error)
	updateMenuItemFn       func(ctx context.Context, m *entity.MenuItem) error
	toggleMenuItemStatusFn func(ctx context.Context, id string, status string) error
	deleteMenuItemFn       func(ctx context.Context, id string) error
}

func (m *mockCatalogRepo) GetSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string) (*entity.MenuItemSlotQuota, error) {
	if m.getSlotQuotaFn != nil {
		return m.getSlotQuotaFn(ctx, menuItemID, date, cutoffID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) UpsertSlotQuota(ctx context.Context, q *entity.MenuItemSlotQuota) error {
	if m.upsertSlotQuotaFn != nil {
		return m.upsertSlotQuotaFn(ctx, q)
	}
	return nil
}

func (m *mockCatalogRepo) DecrementSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error {
	if m.decrementSlotQuotaFn != nil {
		return m.decrementSlotQuotaFn(ctx, menuItemID, date, cutoffID, qty)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) RestoreSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error {
	if m.restoreSlotQuotaFn != nil {
		return m.restoreSlotQuotaFn(ctx, menuItemID, date, cutoffID, qty)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) ListSlotQuotas(ctx context.Context, menuItemID string, date time.Time) ([]*entity.MenuItemSlotQuota, error) {
	if m.listSlotQuotasFn != nil {
		return m.listSlotQuotasFn(ctx, menuItemID, date)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) CreateCategory(ctx context.Context, c *entity.Category) error {
	if m.createCategoryFn != nil {
		return m.createCategoryFn(ctx, c)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) GetCategory(ctx context.Context, id string) (*entity.Category, error) {
	if m.getCategoryFn != nil {
		return m.getCategoryFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error) {
	if m.listCategoriesFn != nil {
		return m.listCategoriesFn(ctx, storeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) UpdateCategory(ctx context.Context, c *entity.Category) error {
	if m.updateCategoryFn != nil {
		return m.updateCategoryFn(ctx, c)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) DeleteCategory(ctx context.Context, id string) error {
	if m.deleteCategoryFn != nil {
		return m.deleteCategoryFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) CreateMenuItem(ctx context.Context, item *entity.MenuItem) error {
	if m.createMenuItemFn != nil {
		return m.createMenuItemFn(ctx, item)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error) {
	if m.getMenuItemFn != nil {
		return m.getMenuItemFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) ListMenuItems(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error) {
	if m.listMenuItemsFn != nil {
		return m.listMenuItemsFn(ctx, storeID, categoryID, status)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCatalogRepo) UpdateMenuItem(ctx context.Context, item *entity.MenuItem) error {
	if m.updateMenuItemFn != nil {
		return m.updateMenuItemFn(ctx, item)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) ToggleMenuItemStatus(ctx context.Context, id string, status string) error {
	if m.toggleMenuItemStatusFn != nil {
		return m.toggleMenuItemStatusFn(ctx, id, status)
	}
	return errors.New("not implemented")
}

func (m *mockCatalogRepo) DeleteMenuItem(ctx context.Context, id string) error {
	if m.deleteMenuItemFn != nil {
		return m.deleteMenuItemFn(ctx, id)
	}
	return errors.New("not implemented")
}

// Stub remaining CatalogRepository methods
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

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestQuotaUsecase_DecrementSlot_Success(t *testing.T) {
	ctx := context.Background()
	date := time.Now()
	menuItemID := "item-001"
	cutoffID := "cutoff-001"

	catalogRepo := &mockCatalogRepo{
		decrementSlotQuotaFn: func(_ context.Context, id string, d time.Time, cid string, qty int) error {
			if id == menuItemID && d.Equal(date) && cid == cutoffID && qty == 2 {
				return nil
			}
			return errors.New("unexpected args")
		},
	}

	uc := NewQuotaUsecase(catalogRepo)
	err := uc.DecrementSlot(ctx, menuItemID, date, cutoffID, 2)

	if err != nil {
		t.Fatalf("DecrementSlot: %v", err)
	}
}

func TestQuotaUsecase_DecrementSlot_ExceededQuota(t *testing.T) {
	ctx := context.Background()
	date := time.Now()
	menuItemID := "item-001"
	cutoffID := "cutoff-001"

	catalogRepo := &mockCatalogRepo{
		decrementSlotQuotaFn: func(_ context.Context, id string, d time.Time, cid string, qty int) error {
			return repository.ErrQuotaExceeded
		},
	}

	uc := NewQuotaUsecase(catalogRepo)
	err := uc.DecrementSlot(ctx, menuItemID, date, cutoffID, 5)

	if !errors.Is(err, ErrQuotaExceeded) {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestQuotaUsecase_RestoreSlot_Success(t *testing.T) {
	ctx := context.Background()
	date := time.Now()
	menuItemID := "item-001"
	cutoffID := "cutoff-001"

	catalogRepo := &mockCatalogRepo{
		restoreSlotQuotaFn: func(_ context.Context, id string, d time.Time, cid string, qty int) error {
			if id == menuItemID && d.Equal(date) && cid == cutoffID && qty == 3 {
				return nil
			}
			return errors.New("unexpected args")
		},
	}

	uc := NewQuotaUsecase(catalogRepo)
	err := uc.RestoreSlot(ctx, menuItemID, date, cutoffID, 3)

	if err != nil {
		t.Fatalf("RestoreSlot: %v", err)
	}
}

func TestQuotaUsecase_RestoreSlot_ClampsToZero(t *testing.T) {
	// RestoreSlot should clamp sold_count to 0; test that repo is called
	ctx := context.Background()
	date := time.Now()
	menuItemID := "item-001"
	cutoffID := "cutoff-001"
	restoreCalled := false

	catalogRepo := &mockCatalogRepo{
		restoreSlotQuotaFn: func(_ context.Context, id string, d time.Time, cid string, qty int) error {
			restoreCalled = true
			return nil
		},
	}

	uc := NewQuotaUsecase(catalogRepo)
	err := uc.RestoreSlot(ctx, menuItemID, date, cutoffID, 100)

	if err != nil {
		t.Fatalf("RestoreSlot: %v", err)
	}
	if !restoreCalled {
		t.Error("expected RestoreSlot to call repo")
	}
}
