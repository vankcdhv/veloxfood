package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// InvitationRepository is the data-access contract for vendor invitations.
type InvitationRepository interface {
	Create(ctx context.Context, inv *entity.Invitation) error
	// AcceptAtomic atomically claims a pending, non-expired invitation.
	// Uses UPDATE...WHERE status='pending' AND expires_at>NOW() RETURNING to ensure only one winner.
	// Returns ErrInvitationInvalidOrExpired (via rows-affected check) if no row matches.
	AcceptAtomic(ctx context.Context, tokenHash, acceptedByUserID string) (*entity.Invitation, error)
	GetByID(ctx context.Context, id string) (*entity.Invitation, error)
	ListByVendor(ctx context.Context, vendorID string, status *entity.InvitationStatus) ([]*entity.Invitation, error)
	FindPendingByVendorAndEmail(ctx context.Context, vendorID, email string) (*entity.Invitation, error)
	Revoke(ctx context.Context, id string) error
}
