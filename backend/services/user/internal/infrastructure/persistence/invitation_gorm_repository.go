package persistence

import (
	"context"
	"errors"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type invitationGormRepository struct {
	db *gorm.DB
}

// NewInvitationGormRepository returns an InvitationRepository backed by GORM.
func NewInvitationGormRepository(db *gorm.DB) repository.InvitationRepository {
	return &invitationGormRepository{db: db}
}

func (r *invitationGormRepository) Create(ctx context.Context, inv *entity.Invitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

// AcceptAtomic atomically marks a pending non-expired invitation as accepted.
// Raw SQL UPDATE...RETURNING ensures only one concurrent accept wins (race-safe).
func (r *invitationGormRepository) AcceptAtomic(ctx context.Context, tokenHash, acceptedByUserID string) (*entity.Invitation, error) {
	var inv entity.Invitation
	result := r.db.WithContext(ctx).Raw(
		`UPDATE invitations
		    SET status = 'accepted', accepted_by_user_id = ?
		  WHERE token_hash = ?
		    AND status = 'pending'
		    AND expires_at > NOW()
		  RETURNING *`,
		acceptedByUserID, tokenHash,
	).Scan(&inv)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		// Token not found, already accepted, or expired — return sentinel error.
		return nil, gorm.ErrRecordNotFound
	}
	return &inv, nil
}

func (r *invitationGormRepository) GetByID(ctx context.Context, id string) (*entity.Invitation, error) {
	var inv entity.Invitation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invitationGormRepository) ListByVendor(ctx context.Context, vendorID string, status *entity.InvitationStatus) ([]*entity.Invitation, error) {
	q := r.db.WithContext(ctx).Where("vendor_id = ?", vendorID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var invitations []*entity.Invitation
	if err := q.Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

func (r *invitationGormRepository) FindPendingByVendorAndEmail(ctx context.Context, vendorID, email string) (*entity.Invitation, error) {
	var inv entity.Invitation
	err := r.db.WithContext(ctx).
		Where("vendor_id = ? AND email = ? AND status = ?", vendorID, email, entity.InvitationStatusPending).
		First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *invitationGormRepository) Revoke(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Invitation{}).
		Where("id = ? AND status = ?", id, entity.InvitationStatusPending).
		Update("status", entity.InvitationStatusRevoked).Error
}
