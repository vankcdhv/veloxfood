package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/services/order/internal/entity"
	"project/services/order/internal/repository"
)

// CartUsecase manages shopping cart operations.
type CartUsecase interface {
	// GetCart returns the customer's cart for a specific store, or nil if none.
	GetCart(ctx context.Context, customerID, storeID string) (*entity.Cart, error)

	// AddOrUpdateItem adds a menu item to the cart or updates its quantity.
	// Cross-store: rejected when customer already has a cart at a different store.
	AddOrUpdateItem(ctx context.Context, req AddCartItemRequest) (*entity.Cart, error)

	// RemoveItem removes a single item from the cart.
	RemoveItem(ctx context.Context, customerID, storeID, menuItemID string) error

	// ClearCart removes all items from the cart (cart row is kept).
	ClearCart(ctx context.Context, customerID, storeID string) error
}

// AddCartItemRequest carries validated input for adding/updating a cart item.
type AddCartItemRequest struct {
	CustomerID      string
	StoreID         string
	MenuItemID      string
	NameSnapshot    string
	PriceSnapshot   int64
	Qty             int
	OptionsSnapshot json.RawMessage
}

type cartUsecase struct {
	cartRepo repository.CartRepository
}

// NewCartUsecase constructs a CartUsecase.
func NewCartUsecase(cartRepo repository.CartRepository) CartUsecase {
	return &cartUsecase{cartRepo: cartRepo}
}

func (uc *cartUsecase) GetCart(ctx context.Context, customerID, storeID string) (*entity.Cart, error) {
	return uc.cartRepo.GetByCustomerAndStore(ctx, customerID, storeID)
}

func (uc *cartUsecase) AddOrUpdateItem(ctx context.Context, req AddCartItemRequest) (*entity.Cart, error) {
	// Enforce single-store constraint: customer may not mix stores in one session.
	existing, err := uc.cartRepo.GetByCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if c.StoreID != req.StoreID && len(c.Items) > 0 {
			return nil, ErrCartCrossStore
		}
	}

	cart, err := uc.cartRepo.GetOrCreate(ctx, req.CustomerID, req.StoreID)
	if err != nil {
		return nil, err
	}

	opts := req.OptionsSnapshot
	if opts == nil {
		opts = json.RawMessage("[]")
	}

	item := &entity.CartItem{
		CartID:          cart.ID,
		MenuItemID:      req.MenuItemID,
		NameSnapshot:    req.NameSnapshot,
		PriceSnapshot:   req.PriceSnapshot,
		Qty:             req.Qty,
		OptionsSnapshot: opts,
	}

	if err := uc.cartRepo.UpsertItem(ctx, item); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "cart item upserted",
		"customer_id", req.CustomerID, "store_id", req.StoreID, "menu_item_id", req.MenuItemID, "qty", req.Qty)

	return uc.cartRepo.GetByCustomerAndStore(ctx, req.CustomerID, req.StoreID)
}

func (uc *cartUsecase) RemoveItem(ctx context.Context, customerID, storeID, menuItemID string) error {
	cart, err := uc.cartRepo.GetByCustomerAndStore(ctx, customerID, storeID)
	if err != nil {
		return err
	}
	if cart == nil {
		return nil // nothing to remove
	}
	return uc.cartRepo.RemoveItem(ctx, cart.ID, menuItemID)
}

func (uc *cartUsecase) ClearCart(ctx context.Context, customerID, storeID string) error {
	cart, err := uc.cartRepo.GetByCustomerAndStore(ctx, customerID, storeID)
	if err != nil {
		return err
	}
	if cart == nil {
		return nil
	}
	return uc.cartRepo.ClearCart(ctx, cart.ID)
}
