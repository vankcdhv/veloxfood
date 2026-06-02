package repository

import (
	"context"

	"project/services/store/internal/entity"

	"gorm.io/gorm"
)

// ShippingRepository manages ShipFeeRules, OperatingHours, and
// OperatingHoursChangeRequests.
type ShippingRepository interface {
	// ---- ShipFeeRules ----

	CreateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error
	GetShipFeeRule(ctx context.Context, id string) (*entity.ShipFeeRule, error)
	ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error)
	UpdateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error
	DeleteShipFeeRule(ctx context.Context, id string) error // soft-delete

	// ResolveShipFee returns the most specific matching fee rule for a store and
	// ordered candidate (scope, id) pairs.  scopes[i] and ids[i] form a pair;
	// the first DB match wins (most-specific first).  Returns nil when no rule
	// matches (store does not serve the location).
	// Example for a ROOM delivery: scopes=["room","floor","building"], ids=[roomID,floorID,buildingID].
	ResolveShipFee(ctx context.Context, storeID string, scopes []string, ids []string) (*entity.ShipFeeRule, error)

	// ---- OperatingHours ----

	CreateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error
	GetOperatingHours(ctx context.Context, id string) (*entity.OperatingHours, error)
	ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error)
	UpdateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error
	DeleteOperatingHours(ctx context.Context, id string) error // soft-delete

	// ---- Bulk replace (used by ApproveHoursChange transaction) ----

	// DeleteOperatingHoursByStore hard-deletes all operating_hours rows for a store within tx.
	DeleteOperatingHoursByStore(ctx context.Context, tx *gorm.DB, storeID string) error

	// BulkCreateOperatingHours inserts a batch of operating_hours rows within tx.
	BulkCreateOperatingHours(ctx context.Context, tx *gorm.DB, rows []*entity.OperatingHours) error

	// ---- OperatingHoursChangeRequests ----

	CreateChangeRequest(ctx context.Context, req *entity.OperatingHoursChangeRequest) error
	GetChangeRequest(ctx context.Context, id string) (*entity.OperatingHoursChangeRequest, error)

	// ListChangeRequests returns requests for a store, optionally filtered by status.
	// Pass status="" to return all statuses.
	ListChangeRequests(ctx context.Context, storeID string, status string) ([]*entity.OperatingHoursChangeRequest, error)

	// UpdateChangeRequestStatus sets status and reviewed_by on a change request.
	UpdateChangeRequestStatus(ctx context.Context, id string, status string, reviewedBy string) error
}
