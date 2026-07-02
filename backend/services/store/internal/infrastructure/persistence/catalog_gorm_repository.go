package persistence

import (
	"context"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


type catalogGormRepository struct {
	db *gorm.DB
}

// NewCatalogGormRepository returns a repository.CatalogRepository backed by GORM.
func NewCatalogGormRepository(db *gorm.DB) repository.CatalogRepository {
	return &catalogGormRepository{db: db}
}

// ---- Categories ----

func (r *catalogGormRepository) CreateCategory(ctx context.Context, c *entity.Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *catalogGormRepository) GetCategory(ctx context.Context, id string) (*entity.Category, error) {
	var c entity.Category
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCategoryByStore fetches a category only if it belongs to storeID.
// Returns gorm.ErrRecordNotFound when the category does not exist or belongs to a different store,
// which maps to 404 — preventing cross-tenant reads from leaking data.
func (r *catalogGormRepository) GetCategoryByStore(ctx context.Context, id, storeID string) (*entity.Category, error) {
	var c entity.Category
	if err := r.db.WithContext(ctx).First(&c, "id = ? AND store_id = ?", id, storeID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *catalogGormRepository) ListCategories(ctx context.Context, storeID string) ([]*entity.Category, error) {
	var rows []*entity.Category
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("sort_order ASC, name ASC").
		Find(&rows).Error
}

func (r *catalogGormRepository) UpdateCategory(ctx context.Context, c *entity.Category) error {
	return r.db.WithContext(ctx).Model(c).Updates(map[string]any{
		"name":       c.Name,
		"sort_order": c.SortOrder,
	}).Error
}

func (r *catalogGormRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Category{}, "id = ?", id).Error
}

// DeleteCategoryByStore soft-deletes the category only when it belongs to storeID.
// If the row does not exist or belongs to a different store, the delete is a no-op
// and the usecase treats RowsAffected==0 as not-found (returns 404).
func (r *catalogGormRepository) DeleteCategoryByStore(ctx context.Context, id, storeID string) error {
	res := r.db.WithContext(ctx).Delete(&entity.Category{}, "id = ? AND store_id = ?", id, storeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---- MenuItems ----

func (r *catalogGormRepository) CreateMenuItem(ctx context.Context, m *entity.MenuItem) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *catalogGormRepository) GetMenuItem(ctx context.Context, id string) (*entity.MenuItem, error) {
	var m entity.MenuItem
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMenuItemByStore fetches a menu item only if it belongs to storeID.
// Returns gorm.ErrRecordNotFound when the item does not exist or belongs to a different store,
// which maps to 404 — preventing cross-tenant reads from leaking data.
func (r *catalogGormRepository) GetMenuItemByStore(ctx context.Context, id, storeID string) (*entity.MenuItem, error) {
	var m entity.MenuItem
	if err := r.db.WithContext(ctx).First(&m, "id = ? AND store_id = ?", id, storeID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *catalogGormRepository) ListMenuItems(ctx context.Context, storeID, categoryID, status string) ([]*entity.MenuItem, error) {
	q := r.db.WithContext(ctx).Where("store_id = ?", storeID).Order("name ASC")
	if categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []*entity.MenuItem
	return rows, q.Find(&rows).Error
}

func (r *catalogGormRepository) UpdateMenuItem(ctx context.Context, m *entity.MenuItem) error {
	return r.db.WithContext(ctx).Model(m).Updates(map[string]any{
		"category_id": m.CategoryID,
		"name":        m.Name,
		"description": m.Description,
		"price":       m.Price,
		"image_url":   m.ImageURL,
		"status":      m.Status,
		"tags":        m.Tags,
	}).Error
}

func (r *catalogGormRepository) ToggleMenuItemStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).
		Model(&entity.MenuItem{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ToggleMenuItemStatusByStore sets only the status column scoped to storeID.
// If the item does not belong to storeID, RowsAffected==0 and gorm.ErrRecordNotFound is returned.
func (r *catalogGormRepository) ToggleMenuItemStatusByStore(ctx context.Context, id, storeID, status string) error {
	res := r.db.WithContext(ctx).
		Model(&entity.MenuItem{}).
		Where("id = ? AND store_id = ?", id, storeID).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *catalogGormRepository) DeleteMenuItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.MenuItem{}, "id = ?", id).Error
}

// DeleteMenuItemByStore soft-deletes the menu item only when it belongs to storeID.
// RowsAffected==0 means the item does not exist or belongs to a different store; returns gorm.ErrRecordNotFound.
func (r *catalogGormRepository) DeleteMenuItemByStore(ctx context.Context, id, storeID string) error {
	res := r.db.WithContext(ctx).Delete(&entity.MenuItem{}, "id = ? AND store_id = ?", id, storeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---- OptionGroups ----

func (r *catalogGormRepository) CreateOptionGroup(ctx context.Context, og *entity.OptionGroup) error {
	return r.db.WithContext(ctx).Create(og).Error
}

func (r *catalogGormRepository) GetOptionGroup(ctx context.Context, id string) (*entity.OptionGroup, error) {
	var og entity.OptionGroup
	if err := r.db.WithContext(ctx).First(&og, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &og, nil
}

// GetOptionGroupByStore fetches an option group only if it belongs to storeID.
// Returns gorm.ErrRecordNotFound when the group does not exist or belongs to a different store,
// which maps to 404 — preventing cross-tenant reads from leaking data.
func (r *catalogGormRepository) GetOptionGroupByStore(ctx context.Context, id, storeID string) (*entity.OptionGroup, error) {
	var og entity.OptionGroup
	if err := r.db.WithContext(ctx).First(&og, "id = ? AND store_id = ?", id, storeID).Error; err != nil {
		return nil, err
	}
	return &og, nil
}

func (r *catalogGormRepository) ListOptionGroups(ctx context.Context, storeID string) ([]*entity.OptionGroup, error) {
	var rows []*entity.OptionGroup
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("name ASC").
		Find(&rows).Error
}

func (r *catalogGormRepository) UpdateOptionGroup(ctx context.Context, og *entity.OptionGroup) error {
	return r.db.WithContext(ctx).Model(og).Updates(map[string]any{
		"name":       og.Name,
		"min_select": og.MinSelect,
		"max_select": og.MaxSelect,
		"required":   og.Required,
	}).Error
}

func (r *catalogGormRepository) DeleteOptionGroup(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.OptionGroup{}, "id = ?", id).Error
}

// DeleteOptionGroupByStore soft-deletes the option group only when it belongs to storeID.
// RowsAffected==0 means the group does not exist or belongs to a different store; returns gorm.ErrRecordNotFound.
func (r *catalogGormRepository) DeleteOptionGroupByStore(ctx context.Context, id, storeID string) error {
	res := r.db.WithContext(ctx).Delete(&entity.OptionGroup{}, "id = ? AND store_id = ?", id, storeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---- Options ----

func (r *catalogGormRepository) CreateOption(ctx context.Context, o *entity.Option) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *catalogGormRepository) GetOption(ctx context.Context, id string) (*entity.Option, error) {
	var o entity.Option
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *catalogGormRepository) ListOptions(ctx context.Context, optionGroupID string) ([]*entity.Option, error) {
	var rows []*entity.Option
	return rows, r.db.WithContext(ctx).
		Where("option_group_id = ?", optionGroupID).
		Order("name ASC").
		Find(&rows).Error
}

func (r *catalogGormRepository) UpdateOption(ctx context.Context, o *entity.Option) error {
	return r.db.WithContext(ctx).Model(o).Updates(map[string]any{
		"name":        o.Name,
		"extra_price": o.ExtraPrice,
	}).Error
}

func (r *catalogGormRepository) DeleteOption(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Option{}, "id = ?", id).Error
}

// ---- MenuItemOptions (junction) ----

func (r *catalogGormRepository) AttachOptionGroup(ctx context.Context, mio *entity.MenuItemOption) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(mio).Error
}

func (r *catalogGormRepository) DetachOptionGroup(ctx context.Context, menuItemID, optionGroupID string) error {
	return r.db.WithContext(ctx).
		Where("menu_item_id = ? AND option_group_id = ?", menuItemID, optionGroupID).
		Delete(&entity.MenuItemOption{}).Error
}

func (r *catalogGormRepository) ListOptionGroupsForItem(ctx context.Context, menuItemID string) ([]*entity.OptionGroup, error) {
	var rows []*entity.OptionGroup
	err := r.db.WithContext(ctx).
		Joins("JOIN menu_item_options mio ON mio.option_group_id = option_groups.id").
		Where("mio.menu_item_id = ?", menuItemID).
		Order("option_groups.name ASC").
		Find(&rows).Error
	return rows, err
}

// ---- Combos ----

func (r *catalogGormRepository) CreateCombo(ctx context.Context, c *entity.Combo) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *catalogGormRepository) GetCombo(ctx context.Context, id string) (*entity.Combo, error) {
	var c entity.Combo
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetComboByStore fetches a combo only if it belongs to storeID.
// Returns gorm.ErrRecordNotFound when the combo does not exist or belongs to a different store,
// which maps to 404 — preventing cross-tenant reads from leaking data.
func (r *catalogGormRepository) GetComboByStore(ctx context.Context, id, storeID string) (*entity.Combo, error) {
	var c entity.Combo
	if err := r.db.WithContext(ctx).First(&c, "id = ? AND store_id = ?", id, storeID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *catalogGormRepository) ListCombos(ctx context.Context, storeID string) ([]*entity.Combo, error) {
	var rows []*entity.Combo
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("name ASC").
		Find(&rows).Error
}

func (r *catalogGormRepository) UpdateCombo(ctx context.Context, c *entity.Combo) error {
	return r.db.WithContext(ctx).Model(c).Updates(map[string]any{
		"name":  c.Name,
		"price": c.Price,
	}).Error
}

func (r *catalogGormRepository) DeleteCombo(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Combo{}, "id = ?", id).Error
}

// DeleteComboByStore soft-deletes the combo only when it belongs to storeID.
// RowsAffected==0 means the combo does not exist or belongs to a different store; returns gorm.ErrRecordNotFound.
func (r *catalogGormRepository) DeleteComboByStore(ctx context.Context, id, storeID string) error {
	res := r.db.WithContext(ctx).Delete(&entity.Combo{}, "id = ? AND store_id = ?", id, storeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---- ComboItems ----

func (r *catalogGormRepository) AddComboItem(ctx context.Context, ci *entity.ComboItem) error {
	return r.db.WithContext(ctx).Create(ci).Error
}

func (r *catalogGormRepository) RemoveComboItem(ctx context.Context, comboID, menuItemID string) error {
	return r.db.WithContext(ctx).
		Where("combo_id = ? AND menu_item_id = ?", comboID, menuItemID).
		Delete(&entity.ComboItem{}).Error
}

func (r *catalogGormRepository) ListComboItems(ctx context.Context, comboID string) ([]*entity.ComboItem, error) {
	var rows []*entity.ComboItem
	return rows, r.db.WithContext(ctx).
		Where("combo_id = ?", comboID).
		Find(&rows).Error
}

// SearchMenuItems performs accent-insensitive fuzzy search on menu item names
// across all non-deleted stores with status='on'. Default order is trigram
// similarity descending; the filter can narrow by price/store and reorder by
// price.
func (r *catalogGormRepository) SearchMenuItems(ctx context.Context, f repository.SearchMenuItemsFilter, limit, offset int) ([]repository.SearchMenuItemRow, int64, error) {
	where := `menu_items.deleted_at IS NULL
	  AND menu_items.status = 'on'
	  AND stores.deleted_at IS NULL
	  AND f_unaccent(lower(menu_items.name)) ILIKE '%' || f_unaccent(lower(?)) || '%'`
	args := []any{f.Q}
	if f.PriceMin > 0 {
		where += " AND menu_items.price >= ?"
		args = append(args, f.PriceMin)
	}
	if f.PriceMax > 0 {
		where += " AND menu_items.price <= ?"
		args = append(args, f.PriceMax)
	}
	if f.StoreID != "" {
		where += " AND stores.id = ?"
		args = append(args, f.StoreID)
	}

	// Total matches (for pagination) — same WHERE as the page query.
	var total int64
	if err := r.db.WithContext(ctx).Raw(`
		SELECT count(*)
		FROM menu_items
		JOIN stores ON stores.id = menu_items.store_id
		WHERE `+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// ORDER BY comes from a fixed set — user input is never interpolated.
	orderBy := ""
	pageArgs := append([]any{}, args...)
	switch f.Sort {
	case "price_asc":
		orderBy = "menu_items.price ASC, menu_items.name"
	case "price_desc":
		orderBy = "menu_items.price DESC, menu_items.name"
	default:
		orderBy = "similarity(f_unaccent(lower(menu_items.name)), f_unaccent(lower(?))) DESC, menu_items.name"
		pageArgs = append(pageArgs, f.Q)
	}
	pageArgs = append(pageArgs, limit, offset)

	var rows []repository.SearchMenuItemRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			menu_items.id          AS id,
			menu_items.name        AS name,
			menu_items.price       AS price,
			menu_items.image_url   AS image_url,
			menu_items.description AS description,
			stores.id              AS store_id,
			stores.name            AS store_name,
			stores.sale_status     AS sale_status
		FROM menu_items
		JOIN stores ON stores.id = menu_items.store_id
		WHERE `+where+`
		ORDER BY `+orderBy+`
		LIMIT ? OFFSET ?
	`, pageArgs...).Scan(&rows).Error
	return rows, total, err
}
