package repository

import (
	"context"

	"project/services/store/internal/entity"
)

// StoreRepository manages the Store aggregate.
type StoreRepository interface {
	// Create persists a new store record.
	Create(ctx context.Context, s *entity.Store) error

	// GetByID returns the store with the given ID, or gorm.ErrRecordNotFound.
	GetByID(ctx context.Context, id string) (*entity.Store, error)

	// GetByVendorID returns the active store linked to the given vendor_id.
	GetByVendorID(ctx context.Context, vendorID string) (*entity.Store, error)

	// List returns all non-deleted stores. Pass saleStatus="" to skip filter.
	List(ctx context.Context, saleStatus string) ([]*entity.Store, error)

	// ListPaged returns a page of non-deleted stores plus the total match count.
	// saleStatus="" skips the status filter; q="" skips the name filter
	// (accent-insensitive substring match). limit<=0 returns all rows.
	ListPaged(ctx context.Context, saleStatus, q string, limit, offset int) ([]*entity.Store, int64, error)

	// ListByOwner returns all non-deleted stores owned by ownerUserID, ordered by name.
	ListByOwner(ctx context.Context, ownerUserID string) ([]*entity.Store, error)

	// Update saves mutable fields on an existing store.
	Update(ctx context.Context, s *entity.Store) error

	// UpdateSaleStatus sets only the sale_status column.
	UpdateSaleStatus(ctx context.Context, id string, status string) error

	// UpdatePickupEnabled sets only the pickup_enabled column.
	UpdatePickupEnabled(ctx context.Context, id string, enabled bool) error

	// UpdateAvatarURL sets only the avatar_url column.
	UpdateAvatarURL(ctx context.Context, id string, url string) error

	// Delete soft-deletes the store.
	Delete(ctx context.Context, id string) error
}
