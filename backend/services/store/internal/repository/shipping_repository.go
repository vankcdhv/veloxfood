package repository

import (
	"context"

	"project/services/store/internal/entity"

	"gorm.io/gorm"
)

// ShippingRepository manages ShipFeeRules, OperatingHours, ShipCutoffs,
// and OperatingHoursChangeRequests.
type ShippingRepository interface {
	// ---- ShipFeeRules ----

	CreateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error
	GetShipFeeRule(ctx context.Context, id string) (*entity.ShipFeeRule, error)
	ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error)
	UpdateShipFeeRule(ctx context.Context, r *entity.ShipFeeRule) error
	DeleteShipFeeRule(ctx context.Context, id string) error // soft-delete

	// ResolveShipFee returns the applicable unit fee for a store + location.
	// Resolution order: room-scoped rule → building-scoped rule → nil (not served).
	ResolveShipFee(ctx context.Context, storeID string, buildingID string, roomID string) (*entity.ShipFeeRule, error)

	// ---- OperatingHours ----

	CreateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error
	GetOperatingHours(ctx context.Context, id string) (*entity.OperatingHours, error)
	ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error)
	UpdateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error
	DeleteOperatingHours(ctx context.Context, id string) error // soft-delete

	// ---- ShipCutoffs ----

	CreateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error
	GetShipCutoff(ctx context.Context, id string) (*entity.ShipCutoff, error)
	ListShipCutoffs(ctx context.Context, storeID string) ([]*entity.ShipCutoff, error)
	UpdateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error
	DeleteShipCutoff(ctx context.Context, id string) error // soft-delete

	// ---- Bulk replace (used by ApproveHoursChange transaction) ----

	// DeleteOperatingHoursByStore hard-deletes all operating_hours rows for a store within tx.
	DeleteOperatingHoursByStore(ctx context.Context, tx *gorm.DB, storeID string) error

	// BulkCreateOperatingHours inserts a batch of operating_hours rows within tx.
	BulkCreateOperatingHours(ctx context.Context, tx *gorm.DB, rows []*entity.OperatingHours) error

	// DeleteShipCutoffsByStore hard-deletes all ship_cutoffs rows for a store within tx.
	DeleteShipCutoffsByStore(ctx context.Context, tx *gorm.DB, storeID string) error

	// BulkCreateShipCutoffs inserts a batch of ship_cutoffs rows within tx.
	BulkCreateShipCutoffs(ctx context.Context, tx *gorm.DB, rows []*entity.ShipCutoff) error

	// ---- OperatingHoursChangeRequests ----

	CreateChangeRequest(ctx context.Context, req *entity.OperatingHoursChangeRequest) error
	GetChangeRequest(ctx context.Context, id string) (*entity.OperatingHoursChangeRequest, error)

	// ListChangeRequests returns requests for a store, optionally filtered by status.
	// Pass status="" to return all statuses.
	ListChangeRequests(ctx context.Context, storeID string, status string) ([]*entity.OperatingHoursChangeRequest, error)

	// UpdateChangeRequestStatus sets status and reviewed_by on a change request.
	UpdateChangeRequestStatus(ctx context.Context, id string, status string, reviewedBy string) error
}
