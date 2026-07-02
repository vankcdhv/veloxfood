package repository

import (
	"context"

	"project/services/store/internal/entity"
)

// FavoriteRepository persists customer bookmarks of stores and menu items.
type FavoriteRepository interface {
	// Add inserts the bookmark; adding an existing one is a no-op (idempotent).
	Add(ctx context.Context, f *entity.Favorite) error
	// Remove deletes the bookmark; removing a missing one is a no-op.
	Remove(ctx context.Context, userID string, targetType entity.FavoriteTargetType, targetID string) error
	// IDs returns the bookmarked target ids of one type for a user.
	IDs(ctx context.Context, userID string, targetType entity.FavoriteTargetType) ([]string, error)
	// FavoriteStores returns the user's bookmarked stores (most recent first),
	// skipping deleted stores.
	FavoriteStores(ctx context.Context, userID string) ([]*entity.Store, error)
	// FavoriteItems returns the user's bookmarked menu items joined with their
	// store (most recent first), skipping deleted/unsellable items.
	FavoriteItems(ctx context.Context, userID string) ([]SearchMenuItemRow, error)
}
