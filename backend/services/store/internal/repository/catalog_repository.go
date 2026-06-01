package repository

import (
	"context"
	"time"

	"project/services/store/internal/entity"
)

// CatalogRepository manages Categories, MenuItems, Quotas, OptionGroups,
// Options, MenuItemOptions, Combos, and ComboItems.
type CatalogRepository interface {
	// ---- Categories ----

	CreateCategory(ctx context.Context, c *entity.Category) error
	GetCategory(ctx context.Context, id string) (*entity.Category, error)
	ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error)
	UpdateCategory(ctx context.Context, c *entity.Category) error
	DeleteCategory(ctx context.Context, id string) error // soft-delete

	// ---- MenuItems ----

	CreateMenuItem(ctx context.Context, m *entity.MenuItem) error
	GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error)

	// ListMenuItems returns items for a store. Pass categoryID="" to skip filter.
	// Pass status="" to skip filter; use "on"/"off" to filter by status.
	ListMenuItems(ctx context.Context, storeID string, categoryID string, status string) ([]*entity.MenuItem, error)

	UpdateMenuItem(ctx context.Context, m *entity.MenuItem) error

	// ToggleMenuItemStatus sets only the status column ("on" or "off").
	ToggleMenuItemStatus(ctx context.Context, id string, status string) error

	DeleteMenuItem(ctx context.Context, id string) error // soft-delete

	// ---- MenuItemSlotQuota ----

	// GetSlotQuota returns the quota record for a specific (item, date, cutoff) triple.
	GetSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string) (*entity.MenuItemSlotQuota, error)

	// UpsertSlotQuota inserts or updates the quota for a (item, date, cutoff) triple.
	UpsertSlotQuota(ctx context.Context, q *entity.MenuItemSlotQuota) error

	// DecrementSlotQuota atomically increments sold_count by qty only when
	// sold_count + qty <= quota. Returns ErrQuotaExceeded if the guard fails
	// (RowsAffected == 0). Callers should treat this as an out-of-stock signal.
	DecrementSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error

	// RestoreSlotQuota decrements sold_count by qty (order cancellation path),
	// clamping to a minimum of 0.
	RestoreSlotQuota(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error

	// ListSlotQuotas returns all quota rows for an item on a given date.
	ListSlotQuotas(ctx context.Context, menuItemID string, date time.Time) ([]*entity.MenuItemSlotQuota, error)

	// ListSlotQuotasForStore returns all quota rows for every item of a store on a
	// given date (customer storefront uses this to show remaining slots per ca).
	ListSlotQuotasForStore(ctx context.Context, storeID string, date time.Time) ([]*entity.MenuItemSlotQuota, error)

	// ---- OptionGroups ----

	CreateOptionGroup(ctx context.Context, og *entity.OptionGroup) error
	GetOptionGroup(ctx context.Context, id string) (*entity.OptionGroup, error)
	ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error)
	UpdateOptionGroup(ctx context.Context, og *entity.OptionGroup) error
	DeleteOptionGroup(ctx context.Context, id string) error // soft-delete

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
	ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error)
	UpdateCombo(ctx context.Context, c *entity.Combo) error
	DeleteCombo(ctx context.Context, id string) error // soft-delete

	// ---- ComboItems ----

	AddComboItem(ctx context.Context, ci *entity.ComboItem) error
	RemoveComboItem(ctx context.Context, comboID string, menuItemID string) error
	ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error)
}
