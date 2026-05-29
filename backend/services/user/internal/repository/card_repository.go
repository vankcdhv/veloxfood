package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// CardRepository is the persistence contract for card_identifiers.
type CardRepository interface {
	Create(ctx context.Context, card *entity.CardIdentifier) error
	GetByID(ctx context.Context, id string) (*entity.CardIdentifier, error)
	// ListByUser returns cards for a user; if includeRevoked=false, only active cards.
	ListByUser(ctx context.Context, userID string, includeRevoked bool) ([]*entity.CardIdentifier, error)
	// Revoke sets revoked_at = now() for the given card ID.
	Revoke(ctx context.Context, id string) error
	// CheckActiveDuplicate returns true if an active (non-revoked) card with same kind+identifier exists.
	CheckActiveDuplicate(ctx context.Context, kind entity.CardKind, identifier string) (bool, error)
}
