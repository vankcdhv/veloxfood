package repository

import (
	"context"

	"project/services/location/internal/entity"
)

// CustomerLocationRepository manages a customer's saved delivery locations.
type CustomerLocationRepository interface {
	Create(ctx context.Context, l *entity.CustomerLocation) error
	GetByID(ctx context.Context, id string) (*entity.CustomerLocation, error)
	ListByCustomer(ctx context.Context, customerID string) ([]*entity.CustomerLocation, error)
	Delete(ctx context.Context, id, customerID string) error
	// SetDefault clears any existing default then marks id as default, atomically.
	SetDefault(ctx context.Context, id, customerID string) error
}
