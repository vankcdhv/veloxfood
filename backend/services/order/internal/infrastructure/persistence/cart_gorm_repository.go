package persistence

import (
	"context"
	"fmt"

	"project/services/order/internal/entity"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type cartGormRepository struct {
	db *gorm.DB
}

// NewCartGormRepository returns a CartRepository backed by GORM/Postgres.
func NewCartGormRepository(db *gorm.DB) repository.CartRepository {
	return &cartGormRepository{db: db}
}

func (r *cartGormRepository) GetOrCreate(ctx context.Context, customerID, storeID string) (*entity.Cart, error) {
	var cart entity.Cart
	err := r.db.WithContext(ctx).
		Where(entity.Cart{CustomerID: customerID, StoreID: storeID}).
		FirstOrCreate(&cart).Error
	if err != nil {
		return nil, fmt.Errorf("cart get-or-create: %w", err)
	}
	return &cart, nil
}

func (r *cartGormRepository) GetByCustomerAndStore(ctx context.Context, customerID, storeID string) (*entity.Cart, error) {
	var cart entity.Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("customer_id = ? AND store_id = ?", customerID, storeID).
		First(&cart).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}
	return &cart, nil
}

func (r *cartGormRepository) GetByCustomer(ctx context.Context, customerID string) ([]*entity.Cart, error) {
	var carts []*entity.Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("customer_id = ?", customerID).
		Find(&carts).Error
	return carts, err
}

// UpsertItem upserts a cart item by (cart_id, menu_item_id).
// qty == 0 delegates to RemoveItem.
func (r *cartGormRepository) UpsertItem(ctx context.Context, item *entity.CartItem) error {
	if item.Qty == 0 {
		return r.RemoveItem(ctx, item.CartID, item.MenuItemID)
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "cart_id"}, {Name: "menu_item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"qty", "name_snapshot", "price_snapshot",
				"cutoff_id", "date", "options_snapshot", "updated_at",
			}),
		}).
		Create(item).Error
}

func (r *cartGormRepository) RemoveItem(ctx context.Context, cartID, menuItemID string) error {
	return r.db.WithContext(ctx).
		Where("cart_id = ? AND menu_item_id = ?", cartID, menuItemID).
		Delete(&entity.CartItem{}).Error
}

func (r *cartGormRepository) ClearCart(ctx context.Context, cartID string) error {
	return r.db.WithContext(ctx).
		Where("cart_id = ?", cartID).
		Delete(&entity.CartItem{}).Error
}

func (r *cartGormRepository) DeleteCart(ctx context.Context, cartID string) error {
	if err := r.ClearCart(ctx, cartID); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Where("id = ?", cartID).Delete(&entity.Cart{}).Error
}

func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
