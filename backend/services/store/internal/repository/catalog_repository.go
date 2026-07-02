package repository

import (
	"context"

	"project/services/store/internal/entity"
)

// SearchMenuItemRow is the flat projection returned by cross-store menu-item search.
// Fields are PascalCase to match the store service JSON envelope convention.
type SearchMenuItemRow struct {
	ID          string `json:"ID"`
	Name        string `json:"Name"`
	Price       int64  `json:"Price"`
	ImageURL    string `json:"ImageURL"`
	Description string `json:"Description"`
	StoreID     string `json:"StoreID"`
	StoreName   string `json:"StoreName"`
	SaleStatus  string `json:"SaleStatus"`
	// OpenNow is filled by the handler (operating hours + sale status), not SQL.
	OpenNow bool `json:"OpenNow" gorm:"-"`
}

// SearchMenuItemsFilter narrows and orders cross-store menu-item search.
// Zero values mean "no constraint". Sort accepts price_asc | price_desc;
// anything else keeps relevance (trigram similarity) order.
type SearchMenuItemsFilter struct {
	Q        string
	Sort     string
	PriceMin int64
	PriceMax int64
	StoreID  string
}

// CatalogRepository manages Categories, MenuItems, OptionGroups,
// Options, MenuItemOptions, Combos, and ComboItems.
type CatalogRepository interface {
	// ---- Categories ----

	CreateCategory(ctx context.Context, c *entity.Category) error
	GetCategory(ctx context.Context, id string) (*entity.Category, error)
	// GetCategoryByStore fetches a category only if it belongs to storeID (returns ErrRecordNotFound otherwise).
	GetCategoryByStore(ctx context.Context, id, storeID string) (*entity.Category, error)
	ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error)
	UpdateCategory(ctx context.Context, c *entity.Category) error
	DeleteCategory(ctx context.Context, id string) error // soft-delete
	// DeleteCategoryByStore soft-deletes the category only if it belongs to storeID.
	DeleteCategoryByStore(ctx context.Context, id, storeID string) error

	// ---- MenuItems ----

	CreateMenuItem(ctx context.Context, m *entity.MenuItem) error
	GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error)
	// GetMenuItemByStore fetches a menu item only if it belongs to storeID (returns ErrRecordNotFound otherwise).
	GetMenuItemByStore(ctx context.Context, id, storeID string) (*entity.MenuItem, error)

	// ListMenuItems returns items for a store. Pass categoryID="" to skip filter.
	// Pass status="" to skip filter; use "on"/"off" to filter by status.
	ListMenuItems(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error)

	UpdateMenuItem(ctx context.Context, m *entity.MenuItem) error

	// ToggleMenuItemStatus sets only the status column ("on" or "off").
	ToggleMenuItemStatus(ctx context.Context, id string, status string) error
	// ToggleMenuItemStatusByStore sets only the status column scoped to storeID (no-op / ErrRecordNotFound if item does not belong to store).
	ToggleMenuItemStatusByStore(ctx context.Context, id, storeID, status string) error

	DeleteMenuItem(ctx context.Context, id string) error // soft-delete
	// DeleteMenuItemByStore soft-deletes the menu item only if it belongs to storeID.
	DeleteMenuItemByStore(ctx context.Context, id, storeID string) error

	// ---- OptionGroups ----

	CreateOptionGroup(ctx context.Context, og *entity.OptionGroup) error
	GetOptionGroup(ctx context.Context, id string) (*entity.OptionGroup, error)
	// GetOptionGroupByStore fetches an option group only if it belongs to storeID (returns ErrRecordNotFound otherwise).
	GetOptionGroupByStore(ctx context.Context, id, storeID string) (*entity.OptionGroup, error)
	ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error)
	UpdateOptionGroup(ctx context.Context, og *entity.OptionGroup) error
	DeleteOptionGroup(ctx context.Context, id string) error // soft-delete
	// DeleteOptionGroupByStore soft-deletes the option group only if it belongs to storeID.
	DeleteOptionGroupByStore(ctx context.Context, id, storeID string) error

	// ---- Options ----

	CreateOption(ctx context.Context, o *entity.Option) error
	GetOption(ctx context.Context, id string) (*entity.Option, error)
	ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error)
	UpdateOption(ctx context.Context, o *entity.Option) error
	DeleteOption(ctx context.Context, id string) error // soft-delete

	// ---- MenuItemOptions (junction) ----

	// AttachOptionGroup links an OptionGroup to a MenuItem.
	AttachOptionGroup(ctx context.Context, mio *entity.MenuItemOption) error

	// DetachOptionGroup removes the link between a MenuItem and an OptionGroup.
	DetachOptionGroup(ctx context.Context, menuItemID string, optionGroupID string) error

	// ListOptionGroupsForItem returns all OptionGroups linked to a MenuItem.
	ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error)

	// ---- Combos ----

	CreateCombo(ctx context.Context, c *entity.Combo) error
	GetCombo(ctx context.Context, id string) (*entity.Combo, error)
	// GetComboByStore fetches a combo only if it belongs to storeID (returns ErrRecordNotFound otherwise).
	GetComboByStore(ctx context.Context, id, storeID string) (*entity.Combo, error)
	ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error)
	UpdateCombo(ctx context.Context, c *entity.Combo) error
	DeleteCombo(ctx context.Context, id string) error // soft-delete
	// DeleteComboByStore soft-deletes the combo only if it belongs to storeID.
	DeleteComboByStore(ctx context.Context, id, storeID string) error

	// ---- ComboItems ----

	AddComboItem(ctx context.Context, ci *entity.ComboItem) error
	RemoveComboItem(ctx context.Context, comboID string, menuItemID string) error
	ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error)

	// ---- Global search ----

	// SearchMenuItems performs accent-insensitive fuzzy name search across all
	// active stores, paginated. Default order is trigram similarity desc, then
	// name; the filter can narrow by price range / store and reorder by price.
	// Returns the page rows plus the total match count.
	SearchMenuItems(ctx context.Context, f SearchMenuItemsFilter, limit, offset int) ([]SearchMenuItemRow, int64, error)
}
