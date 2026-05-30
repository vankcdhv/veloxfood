package repository

import (
	"context"
	"time"

	"project/services/user/internal/entity"
)

// ShipperProfileRepository persists shipper onboarding profiles.
type ShipperProfileRepository interface {
	Create(ctx context.Context, p *entity.ShipperProfile) error
	GetByUserID(ctx context.Context, userID string) (*entity.ShipperProfile, error)
	// UpdateStatus sets the approval state (+ approver/timestamp for approvals).
	UpdateStatus(ctx context.Context, userID string, status entity.ShipperStatus, approvedBy *string, approvedAt *time.Time) error
	// List returns shipper profiles filtered by optional status (nil = all).
	List(ctx context.Context, status *entity.ShipperStatus, offset, limit int) ([]*entity.ShipperProfile, int64, error)
}
