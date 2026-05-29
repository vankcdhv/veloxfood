package usecase

import (
	"context"
	"time"

	"project/services/user/internal/entity"
)

// InviteStaffInput is the request body for inviting a staff member.
type InviteStaffInput struct {
	Email        string              `json:"email"`
	RoleInVendor entity.RoleInVendor `json:"role_in_vendor"`
	// VendorName is used in the invite email subject/body — caller must supply.
	VendorName string `json:"vendor_name"`
}

// InviteStaffOutput carries the created invitation metadata.
type InviteStaffOutput struct {
	InvitationID string    `json:"invitation_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AcceptInvitationInput holds the raw token from the invite link.
type AcceptInvitationInput struct {
	Token string `json:"token"`
}

// AcceptInvitationOutput is returned after successfully accepting an invitation.
type AcceptInvitationOutput struct {
	VendorID string              `json:"vendor_id"`
	Role     entity.RoleInVendor `json:"role"`
}

// VendorStaffUsecase handles invitation lifecycle and staff membership management.
type VendorStaffUsecase interface {
	Invite(ctx context.Context, ownerUserID, vendorID string, input InviteStaffInput) (*InviteStaffOutput, error)
	Accept(ctx context.Context, userID string, input AcceptInvitationInput) (*AcceptInvitationOutput, error)
	ListStaff(ctx context.Context, vendorID string, status *entity.VendorMembershipStatus) ([]*entity.VendorMembership, error)
	RemoveStaff(ctx context.Context, removerID, vendorID, targetUserID string) error
	ListMyVendors(ctx context.Context, userID string) ([]*entity.VendorMembership, error)
}
