package usecase

import "project/pkg/apperror"

var (
	ErrStoreNotFound         = apperror.NotFound("store not found")
	ErrStoreAlreadyExists    = apperror.Conflict("a store for this vendor already exists")
	ErrCategoryNotFound      = apperror.NotFound("category not found")
	ErrMenuItemNotFound      = apperror.NotFound("menu item not found")
	ErrOptionGroupNotFound   = apperror.NotFound("option group not found")
	ErrOptionNotFound        = apperror.NotFound("option not found")
	ErrComboNotFound         = apperror.NotFound("combo not found")
	ErrShipFeeRuleNotFound   = apperror.NotFound("ship fee rule not found")
	ErrChangeRequestNotFound = apperror.NotFound("hours change request not found")
	ErrLocationNotServed     = apperror.BadRequest("store does not deliver to the requested location")
	ErrQuotaExceeded         = apperror.Conflict("slot quota exceeded for this item")
	ErrForbidden             = apperror.Forbidden("access denied")
	ErrInvalidSaleStatus     = apperror.BadRequest("invalid sale_status; must be OPEN, CLOSED_TODAY, or PAUSED")
	ErrInvalidChangeStatus   = apperror.BadRequest("change request is not in pending status")
)
