package usecase

import "project/pkg/apperror"

var (
	ErrEmailRequired               = apperror.BadRequest("email is required")
	ErrVendorNameRequired          = apperror.BadRequest("vendor_name is required")
	ErrInvitationNotFound          = apperror.NotFound("invitation not found")
	ErrInvitationDuplicate         = apperror.Conflict("pending invitation for this email already exists")
	ErrInvitationInvalidOrExpired  = apperror.Gone("invitation token is invalid, already used, or expired")
	ErrCannotRemoveLastOwner       = apperror.Conflict("cannot remove the last owner of a vendor")
	ErrVendorRoleNotConfigured     = apperror.Internal("vendor role code not found — ask admin to create it via roles API")
	ErrMembershipNotFound          = apperror.NotFound("vendor membership not found")
	ErrCannotRemoveOwnerAsSelf     = apperror.Conflict("vendor owner cannot remove their own owner membership — transfer ownership first")
	ErrInvalidInviteRole           = apperror.BadRequest("role_in_vendor must be one of: MANAGER, KITCHEN, CASHIER")
)
