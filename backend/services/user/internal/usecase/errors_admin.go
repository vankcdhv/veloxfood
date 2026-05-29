package usecase

import "project/pkg/apperror"

var (
	// ErrCannotActOnSelf prevents admin from suspending/removing own SUPER_ADMIN role.
	ErrCannotActOnSelf = apperror.BadRequest("cannot perform this action on yourself")

	// ErrLastSuperAdmin prevents removing the last SUPER_ADMIN role assignment.
	ErrLastSuperAdmin = apperror.BadRequest("cannot remove the last SUPER_ADMIN — system would be locked out")

	// ErrVendorAlreadyProcessed returned when approve/reject is called on an already-processed vendor.
	ErrVendorAlreadyProcessed = apperror.Conflict("vendor has already been approved or rejected")

	// ErrUserAlreadyInStatus returned when suspend/reactivate is called and user is already in target status.
	ErrUserAlreadyInStatus = apperror.Conflict("user is already in the requested status")
)
