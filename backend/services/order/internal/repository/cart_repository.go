package repository

import (
	"context"

	"project/services/order/internal/entity"
)

// CartRepository is the persistence contract for carts and cart items.
type CartRepository interface {
	// GetOrCreate returns the existing cart for (customerID, storeID) or creates one.
	GetOrCreate(ctx context.Context, customerID, storeID string) (*entity.Cart, error)

	// GetByCustomerAndStore returns the cart for (customerID, storeID) with items loaded.
	// Returns nil, nil when no cart exists.
	GetByCustomerAndStore(ctx context.Context, customerID, storeID string) (*entity.Cart, error)

	// GetByCustomer returns all carts for a customer (across stores), items loaded.
	GetByCustomer(ctx context.Context, customerID string) ([]*entity.Cart, error)

	// UpsertItem adds or updates a cart item by (cart_id, menu_item_id).
	// qty=0 is treated as remove.
	UpsertItem(ctx context.Context, item *entity.CartItem) error

	// RemoveItem deletes a cart item by (cartID, menuItemID).
	RemoveItem(ctx context.Context, cartID, menuItemID string) error

	// ClearCart removes all items from the cart (e.g. after order placement).
	ClearCart(ctx context.Context, cartID string) error

	// DeleteCart removes the cart and all its items.
	DeleteCart(ctx context.Context, cartID string) error
}
