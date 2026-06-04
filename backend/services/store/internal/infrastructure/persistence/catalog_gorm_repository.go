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

func (r *catalogGormRepository) DeleteMenuItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.MenuItem{}, "id = ?", id).Error
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
// across all non-deleted stores with status='on'. Ordered by trigram similarity
// descending so the best match appears first.
func (r *catalogGormRepository) SearchMenuItems(ctx context.Context, q string, limit int) ([]repository.SearchMenuItemRow, error) {
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
		WHERE menu_items.deleted_at IS NULL
		  AND menu_items.status = 'on'
		  AND stores.deleted_at IS NULL
		  AND f_unaccent(lower(menu_items.name)) ILIKE '%' || f_unaccent(lower(?)) || '%'
		ORDER BY similarity(f_unaccent(lower(menu_items.name)), f_unaccent(lower(?))) DESC,
		         menu_items.name
		LIMIT ?
	`, q, q, limit).Scan(&rows).Error
	return rows, err
}
