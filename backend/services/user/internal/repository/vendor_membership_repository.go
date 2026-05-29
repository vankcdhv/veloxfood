package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// VendorMembershipRepository is the data-access contract for vendor membership records.
type VendorMembershipRepository interface {
	Create(ctx context.Context, m *entity.VendorMembership) error
	GetByUserAndVendor(ctx context.Context, userID, vendorID string) (*entity.VendorMembership, error)
	ListByVendor(ctx context.Context, vendorID string, status *entity.VendorMembershipStatus) ([]*entity.VendorMembership, error)
	ListByUser(ctx context.Context, userID string) ([]*entity.VendorMembership, error)
	UpdateStatus(ctx context.Context, id string, status entity.VendorMembershipStatus) error
	// UpsertActiveMembership inserts or updates an active membership on (user_id, vendor_id) conflict.
	UpsertActiveMembership(ctx context.Context, m *entity.VendorMembership) error
	// CountOwners returns the number of OWNER members with status=active for a vendor.
	CountOwners(ctx context.Context, vendorID string) (int, error)
}
