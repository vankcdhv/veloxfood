package usecase

import (
	"context"
	"errors"
	"fmt"

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
	UpdateCategory(ctx context.Context, id, name string, sortOrder int) (*entity.Category, error)
	DeleteCategory(ctx context.Context, id string) error

	// Menu items
	CreateMenuItem(ctx context.Context, storeID, categoryID, name, description string, price int64, tags string) (*entity.MenuItem, error)
	GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error)
	ListMenuItems(ctx context.Context, storeID, categoryID, status string) ([]*entity.MenuItem, error)
	UpdateMenuItem(ctx context.Context, id, name, description string, price int64, tags string, imageURL *string) (*entity.MenuItem, error)
	ToggleMenuItemStatus(ctx context.Context, id, status string) error
	SetMenuItemImage(ctx context.Context, id, imageURL string) error
	DeleteMenuItem(ctx context.Context, id string) error

	// Option groups
	CreateOptionGroup(ctx context.Context, storeID, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error)
	ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error)
	UpdateOptionGroup(ctx context.Context, id, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error)
	DeleteOptionGroup(ctx context.Context, id string) error

	// Options
	CreateOption(ctx context.Context, optionGroupID, name string, extraPrice int64) (*entity.Option, error)
	ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error)
	UpdateOption(ctx context.Context, id, name string, extraPrice int64) (*entity.Option, error)
	DeleteOption(ctx context.Context, id string) error

	// MenuItem ↔ OptionGroup links
	AttachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error
	DetachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error
	ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error)

	// Combos
	CreateCombo(ctx context.Context, storeID, name string, price int64) (*entity.Combo, error)
	ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error)
	UpdateCombo(ctx context.Context, id, name string, price int64) (*entity.Combo, error)
	DeleteCombo(ctx context.Context, id string) error
	AddComboItem(ctx context.Context, comboID, menuItemID string, quantity int) error
	RemoveComboItem(ctx context.Context, comboID, menuItemID string) error
	ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error)

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

func (uc *catalogUsecase) UpdateCategory(ctx context.Context, id, name string, sortOrder int) (*entity.Category, error) {
	c, err := uc.catalogRepo.GetCategory(ctx, id)
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

func (uc *catalogUsecase) DeleteCategory(ctx context.Context, id string) error {
	return uc.catalogRepo.DeleteCategory(ctx, id)
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

func (uc *catalogUsecase) UpdateMenuItem(ctx context.Context, id, name, description string, price int64, tags string, imageURL *string) (*entity.MenuItem, error) {
	m, err := uc.catalogRepo.GetMenuItem(ctx, id)
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

func (uc *catalogUsecase) ToggleMenuItemStatus(ctx context.Context, id, status string) error {
	if status != "on" && status != "off" {
		return fmt.Errorf("invalid status: must be 'on' or 'off'")
	}
	return uc.catalogRepo.ToggleMenuItemStatus(ctx, id, status)
}

func (uc *catalogUsecase) SetMenuItemImage(ctx context.Context, id, imageURL string) error {
	m, err := uc.catalogRepo.GetMenuItem(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMenuItemNotFound
	}
	if err != nil {
		return err
	}
	m.ImageURL = imageURL
	return uc.catalogRepo.UpdateMenuItem(ctx, m)
}

func (uc *catalogUsecase) DeleteMenuItem(ctx context.Context, id string) error {
	return uc.catalogRepo.DeleteMenuItem(ctx, id)
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

func (uc *catalogUsecase) UpdateOptionGroup(ctx context.Context, id, name string, minSelect, maxSelect int, required bool) (*entity.OptionGroup, error) {
	og, err := uc.catalogRepo.GetOptionGroup(ctx, id)
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

func (uc *catalogUsecase) DeleteOptionGroup(ctx context.Context, id string) error {
	return uc.catalogRepo.DeleteOptionGroup(ctx, id)
}

// ─── Options ─────────────────────────────────────────────────────────────────

func (uc *catalogUsecase) CreateOption(ctx context.Context, optionGroupID, name string, extraPrice int64) (*entity.Option, error) {
	o := &entity.Option{OptionGroupID: optionGroupID, Name: name, ExtraPrice: extraPrice}
	if err := uc.catalogRepo.CreateOption(ctx, o); err != nil {
		return nil, fmt.Errorf("create option: %w", err)
	}
	return o, nil
}

func (uc *catalogUsecase) ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error) {
	return uc.catalogRepo.ListOptions(ctx, optionGroupID)
}

func (uc *catalogUsecase) UpdateOption(ctx context.Context, id, name string, extraPrice int64) (*entity.Option, error) {
	o, err := uc.catalogRepo.GetOption(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOptionNotFound
	}
	if err != nil {
		return nil, err
	}
	o.Name = name
	o.ExtraPrice = extraPrice
	if err := uc.catalogRepo.UpdateOption(ctx, o); err != nil {
		return nil, fmt.Errorf("update option: %w", err)
	}
	return o, nil
}

func (uc *catalogUsecase) DeleteOption(ctx context.Context, id string) error {
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

func (uc *catalogUsecase) UpdateCombo(ctx context.Context, id, name string, price int64) (*entity.Combo, error) {
	c, err := uc.catalogRepo.GetCombo(ctx, id)
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

func (uc *catalogUsecase) DeleteCombo(ctx context.Context, id string) error {
	return uc.catalogRepo.DeleteCombo(ctx, id)
}

func (uc *catalogUsecase) AddComboItem(ctx context.Context, comboID, menuItemID string, quantity int) error {
	return uc.catalogRepo.AddComboItem(ctx, &entity.ComboItem{
		ComboID:    comboID,
		MenuItemID: menuItemID,
		Quantity:   quantity,
	})
}

func (uc *catalogUsecase) RemoveComboItem(ctx context.Context, comboID, menuItemID string) error {
	return uc.catalogRepo.RemoveComboItem(ctx, comboID, menuItemID)
}

func (uc *catalogUsecase) ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error) {
	return uc.catalogRepo.ListComboItems(ctx, comboID)
}

