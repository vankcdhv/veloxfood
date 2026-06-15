package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// CatalogUsecase manages the full catalog (categories, menu items, option groups,
// options, combos) for a store.
type CatalogUsecase interface {
	// Categories
	CreateCategory(ctx context.Context, storeID, name string, sortOrder int) (*entity.Category, error)
	ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error)
	// UpdateCategory scopes the update to storeID so a vendor cannot mutate another store's category.
	UpdateCategory(ctx context.Context, storeID, id, name string, sortOrder int) (*entity.Category, error)
	// DeleteCategory scopes the delete to storeID so a vendor cannot remove another store's category.
	DeleteCategory(ctx context.Context, storeID, id string) error

	// Menu items
	CreateMenuItem(ctx context.Context, storeID, categoryID, name, description string, price int64, tags string) (*entity.MenuItem, error)
	GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error)
	ListMenuItems(ctx context.Context, storeID, categoryID, status string) ([]*entity.MenuItem, error)
	// UpdateMenuItem scopes the update to storeID to prevent cross-tenant edits.
	UpdateMenuItem(ctx context.Context, storeID, id, name, description string, price int64, tags string, imageURL *string) (*entity.MenuItem, error)
	// ToggleMenuItemStatus scopes the status change to storeID to prevent cross-tenant edits.
	ToggleMenuItemStatus(ctx context.Context, storeID, id, status string) error
	// SetMenuItemImage scopes the image update to storeID to prevent cross-tenant edits.
	SetMenuItemImage(ctx context.Context, storeID, id, imageURL string) error
	// DeleteMenuItem scopes the delete to storeID to prevent cross-tenant edits.
	DeleteMenuItem(ctx context.Context, storeID, id string) error

	// Option groups
	CreateOptionGroup(ctx context.Context, storeID, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error)
	ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error)
	// UpdateOptionGroup scopes the update to storeID to prevent cross-tenant edits.
	UpdateOptionGroup(ctx context.Context, storeID, id, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error)
	// DeleteOptionGroup scopes the delete to storeID to prevent cross-tenant edits.
	DeleteOptionGroup(ctx context.Context, storeID, id string) error

	// Options — Option rows have no direct StoreID; ownership is inherited from their
	// parent OptionGroup. The storeID parameter is asserted against the parent's StoreID.
	// CreateOption verifies the target option group belongs to storeID before inserting.
	CreateOption(ctx context.Context, storeID, optionGroupID, name string, extraPrice int64) (*entity.Option, error)
	ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error)
	// UpdateOption verifies the option's parent group belongs to storeID before updating.
	UpdateOption(ctx context.Context, storeID, id, name string, extraPrice int64) (*entity.Option, error)
	// DeleteOption verifies the option's parent group belongs to storeID before deleting.
	DeleteOption(ctx context.Context, storeID, id string) error

	// MenuItem ↔ OptionGroup links
	AttachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error
	DetachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error
	ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error)

	// Combos
	CreateCombo(ctx context.Context, storeID, name string, price int64) (*entity.Combo, error)
	ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error)
	// UpdateCombo scopes the update to storeID to prevent cross-tenant edits.
	UpdateCombo(ctx context.Context, storeID, id, name string, price int64) (*entity.Combo, error)
	// DeleteCombo scopes the delete to storeID to prevent cross-tenant edits.
	DeleteCombo(ctx context.Context, storeID, id string) error
	// AddComboItem verifies the combo belongs to storeID before adding a line item.
	AddComboItem(ctx context.Context, storeID, comboID, menuItemID string, quantity int) error
	// RemoveComboItem verifies the combo belongs to storeID before removing a line item.
	RemoveComboItem(ctx context.Context, storeID, comboID, menuItemID string) error
	ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error)

	// Global search
	// SearchMenuItems finds sellable items by name (accent-insensitive, fuzzy).
	// Empty q returns an empty slice. limit is clamped to [1, 50]; default 24.
	SearchMenuItems(ctx context.Context, q string, limit int) ([]repository.SearchMenuItemRow, error)
}

type catalogUsecase struct {
	catalogRepo repository.CatalogRepository
}

func NewCatalogUsecase(catalogRepo repository.CatalogRepository) CatalogUsecase {
	return &catalogUsecase{catalogRepo: catalogRepo}
}

// ─── Categories ──────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateCategory(ctx context.Context, storeID, name string, sortOrder int) (*entity.Category, error) {
	c := &entity.Category{StoreID: storeID, Name: name, SortOrder: sortOrder}
	if err := uc.catalogRepo.CreateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return c, nil
}

func (uc *catalogUsecase) ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error) {
	return uc.catalogRepo.ListCategories(ctx, storeID)
}

func (uc *catalogUsecase) UpdateCategory(ctx context.Context, storeID, id, name string, sortOrder int) (*entity.Category, error) {
	// Scope the lookup to storeID so a vendor cannot update another store's category.
	c, err := uc.catalogRepo.GetCategoryByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Name = name
	c.SortOrder = sortOrder
	if err := uc.catalogRepo.UpdateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return c, nil
}

func (uc *catalogUsecase) DeleteCategory(ctx context.Context, storeID, id string) error {
	// Scope the delete to storeID so a vendor cannot remove another store's category.
	err := uc.catalogRepo.DeleteCategoryByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCategoryNotFound
	}
	return err
}

// ─── Menu items ───────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateMenuItem(ctx context.Context, storeID, categoryID, name, description string, price int64, tags string) (*entity.MenuItem, error) {
	m := &entity.MenuItem{
		StoreID:     storeID,
		CategoryID:  categoryID,
		Name:        name,
		Description: description,
		Price:       price,
		Tags:        tags,
		Status:      "on",
	}
	if err := uc.catalogRepo.CreateMenuItem(ctx, m); err != nil {
		return nil, fmt.Errorf("create menu item: %w", err)
	}
	return m, nil
}

func (uc *catalogUsecase) GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error) {
	m, err := uc.catalogRepo.GetMenuItem(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMenuItemNotFound
	}
	return m, err
}

func (uc *catalogUsecase) ListMenuItems(ctx context.Context, storeID, categoryID, status string) ([]*entity.MenuItem, error) {
	return uc.catalogRepo.ListMenuItems(ctx, storeID, categoryID, status)
}

func (uc *catalogUsecase) UpdateMenuItem(ctx context.Context, storeID, id, name, description string, price int64, tags string, imageURL *string) (*entity.MenuItem, error) {
	// Scope the lookup to storeID so a vendor cannot update another store's menu item.
	m, err := uc.catalogRepo.GetMenuItemByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMenuItemNotFound
	}
	if err != nil {
		return nil, err
	}
	m.Name = name
	m.Description = description
	m.Price = price
	m.Tags = tags
	// Only touch the image when the caller explicitly sends image_url.
	if imageURL != nil {
		m.ImageURL = *imageURL
	}
	if err := uc.catalogRepo.UpdateMenuItem(ctx, m); err != nil {
		return nil, fmt.Errorf("update menu item: %w", err)
	}
	return m, nil
}

func (uc *catalogUsecase) ToggleMenuItemStatus(ctx context.Context, storeID, id, status string) error {
	if status != "on" && status != "off" {
		return fmt.Errorf("invalid status: must be 'on' or 'off'")
	}
	// Scope the status change to storeID so a vendor cannot toggle another store's item.
	err := uc.catalogRepo.ToggleMenuItemStatusByStore(ctx, id, storeID, status)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMenuItemNotFound
	}
	return err
}

func (uc *catalogUsecase) SetMenuItemImage(ctx context.Context, storeID, id, imageURL string) error {
	// Scope the lookup to storeID so a vendor cannot set image on another store's item.
	m, err := uc.catalogRepo.GetMenuItemByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMenuItemNotFound
	}
	if err != nil {
		return err
	}
	m.ImageURL = imageURL
	return uc.catalogRepo.UpdateMenuItem(ctx, m)
}

func (uc *catalogUsecase) DeleteMenuItem(ctx context.Context, storeID, id string) error {
	// Scope the delete to storeID so a vendor cannot remove another store's menu item.
	err := uc.catalogRepo.DeleteMenuItemByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMenuItemNotFound
	}
	return err
}

// ─── Option groups ────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateOptionGroup(ctx context.Context, storeID, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error) {
	og := &entity.OptionGroup{StoreID: storeID, Name: name, MinSelect: minSelect, MaxSelect: maxSelect, Required: required}
	if err := uc.catalogRepo.CreateOptionGroup(ctx, og); err != nil {
		return nil, fmt.Errorf("create option group: %w", err)
	}
	return og, nil
}

func (uc *catalogUsecase) ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error) {
	return uc.catalogRepo.ListOptionGroups(ctx, storeID)
}

func (uc *catalogUsecase) UpdateOptionGroup(ctx context.Context, storeID, id, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error) {
	// Scope the lookup to storeID so a vendor cannot update another store's option group.
	og, err := uc.catalogRepo.GetOptionGroupByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOptionGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	og.Name = name
	og.MinSelect = minSelect
	og.MaxSelect = maxSelect
	og.Required = required
	if err := uc.catalogRepo.UpdateOptionGroup(ctx, og); err != nil {
		return nil, fmt.Errorf("update option group: %w", err)
	}
	return og, nil
}

func (uc *catalogUsecase) DeleteOptionGroup(ctx context.Context, storeID, id string) error {
	// Scope the delete to storeID so a vendor cannot remove another store's option group.
	err := uc.catalogRepo.DeleteOptionGroupByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrOptionGroupNotFound
	}
	return err
}

// ─── Options ─────────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateOption(ctx context.Context, storeID, optionGroupID, name string, extraPrice int64) (*entity.Option, error) {
	// Options inherit store ownership from their parent OptionGroup.
	// Verify the parent group belongs to storeID before inserting to prevent cross-tenant writes.
	og, err := uc.catalogRepo.GetOptionGroupByStore(ctx, optionGroupID, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOptionGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	o := &entity.Option{OptionGroupID: og.ID, Name: name, ExtraPrice: extraPrice}
	if err := uc.catalogRepo.CreateOption(ctx, o); err != nil {
		return nil, fmt.Errorf("create option: %w", err)
	}
	return o, nil
}

func (uc *catalogUsecase) ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error) {
	return uc.catalogRepo.ListOptions(ctx, optionGroupID)
}

func (uc *catalogUsecase) UpdateOption(ctx context.Context, storeID, id, name string, extraPrice int64) (*entity.Option, error) {
	// Options have no direct StoreID; verify ownership by loading the option then its parent group.
	o, err := uc.catalogRepo.GetOption(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOptionNotFound
	}
	if err != nil {
		return nil, err
	}
	// Assert the parent option group belongs to the authorised store to prevent cross-tenant edits.
	og, err := uc.catalogRepo.GetOptionGroup(ctx, o.OptionGroupID)
	if err != nil || og.StoreID != storeID {
		return nil, ErrOptionNotFound
	}
	o.Name = name
	o.ExtraPrice = extraPrice
	if err := uc.catalogRepo.UpdateOption(ctx, o); err != nil {
		return nil, fmt.Errorf("update option: %w", err)
	}
	return o, nil
}

func (uc *catalogUsecase) DeleteOption(ctx context.Context, storeID, id string) error {
	// Options have no direct StoreID; verify ownership by loading the option then its parent group.
	o, err := uc.catalogRepo.GetOption(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrOptionNotFound
	}
	if err != nil {
		return err
	}
	// Assert the parent option group belongs to the authorised store to prevent cross-tenant deletes.
	og, err := uc.catalogRepo.GetOptionGroup(ctx, o.OptionGroupID)
	if err != nil || og.StoreID != storeID {
		return ErrOptionNotFound
	}
	return uc.catalogRepo.DeleteOption(ctx, id)
}

// ─── MenuItem ↔ OptionGroup links ─────────────────────────────────────────────

func (uc *catalogUsecase) AttachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error {
	return uc.catalogRepo.AttachOptionGroup(ctx, &entity.MenuItemOption{
		MenuItemID:    menuItemID,
		OptionGroupID: optionGroupID,
	})
}

func (uc *catalogUsecase) DetachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error {
	return uc.catalogRepo.DetachOptionGroup(ctx, menuItemID, optionGroupID)
}

func (uc *catalogUsecase) ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error) {
	return uc.catalogRepo.ListOptionGroupsForItem(ctx, menuItemID)
}

// ─── Combos ───────────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateCombo(ctx context.Context, storeID, name string, price int64) (*entity.Combo, error) {
	c := &entity.Combo{StoreID: storeID, Name: name, Price: price}
	if err := uc.catalogRepo.CreateCombo(ctx, c); err != nil {
		return nil, fmt.Errorf("create combo: %w", err)
	}
	return c, nil
}

func (uc *catalogUsecase) ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error) {
	return uc.catalogRepo.ListCombos(ctx, storeID)
}

func (uc *catalogUsecase) UpdateCombo(ctx context.Context, storeID, id, name string, price int64) (*entity.Combo, error) {
	// Scope the lookup to storeID so a vendor cannot update another store's combo.
	c, err := uc.catalogRepo.GetComboByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrComboNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Name = name
	c.Price = price
	if err := uc.catalogRepo.UpdateCombo(ctx, c); err != nil {
		return nil, fmt.Errorf("update combo: %w", err)
	}
	return c, nil
}

func (uc *catalogUsecase) DeleteCombo(ctx context.Context, storeID, id string) error {
	// Scope the delete to storeID so a vendor cannot remove another store's combo.
	err := uc.catalogRepo.DeleteComboByStore(ctx, id, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrComboNotFound
	}
	return err
}

func (uc *catalogUsecase) AddComboItem(ctx context.Context, storeID, comboID, menuItemID string, quantity int) error {
	// ComboItems inherit store ownership from their parent Combo.
	// Verify the combo belongs to storeID before adding a line item to prevent cross-tenant writes.
	_, err := uc.catalogRepo.GetComboByStore(ctx, comboID, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrComboNotFound
	}
	if err != nil {
		return err
	}
	return uc.catalogRepo.AddComboItem(ctx, &entity.ComboItem{
		ComboID:    comboID,
		MenuItemID: menuItemID,
		Quantity:   quantity,
	})
}

func (uc *catalogUsecase) RemoveComboItem(ctx context.Context, storeID, comboID, menuItemID string) error {
	// Verify the combo belongs to storeID before removing a line item to prevent cross-tenant deletes.
	_, err := uc.catalogRepo.GetComboByStore(ctx, comboID, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrComboNotFound
	}
	if err != nil {
		return err
	}
	return uc.catalogRepo.RemoveComboItem(ctx, comboID, menuItemID)
}

func (uc *catalogUsecase) ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error) {
	return uc.catalogRepo.ListComboItems(ctx, comboID)
}

// ─── Global search ────────────────────────────────────────────────────────────

const (
	searchDefaultLimit = 24
	searchMaxLimit     = 50
)

func (uc *catalogUsecase) SearchMenuItems(ctx context.Context, q string, limit int) ([]repository.SearchMenuItemRow, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []repository.SearchMenuItemRow{}, nil
	}
	if limit <= 0 {
		limit = searchDefaultLimit
	}
	if limit > searchMaxLimit {
		limit = searchMaxLimit
	}
	return uc.catalogRepo.SearchMenuItems(ctx, q, limit)
}

