package usecase

import "project/pkg/apperror"

var (
	ErrPromotionNotFound    = apperror.NotFound("promotion not found")
	ErrPromotionCodeExists  = apperror.Conflict("a promotion with this code already exists for the store")
	ErrPromotionInactive    = apperror.BadRequest("promotion is not active")
	ErrPromotionExpired     = apperror.BadRequest("promotion has expired")
	ErrPromotionNotStarted  = apperror.BadRequest("promotion has not started yet")
	ErrMinOrderNotMet       = apperror.BadRequest("order subtotal does not meet the minimum order requirement")
	ErrUsageLimitReached    = apperror.Conflict("promotion usage limit has been reached")
	ErrWrongStore           = apperror.BadRequest("promotion does not belong to the requested store")
	ErrDuplicateType        = apperror.BadRequest("cannot apply two promotions of the same type to one order")
	ErrStoreNotFound        = apperror.NotFound("store not found")
	ErrForbidden            = apperror.Forbidden("access denied")
	ErrInvalidValueKind     = apperror.BadRequest("value_kind must be PERCENT or AMOUNT")
	ErrInvalidType          = apperror.BadRequest("type must be ORDER_DISCOUNT or SHIP_DISCOUNT")
	ErrInvalidStatus        = apperror.BadRequest("status must be ACTIVE or INACTIVE")
	ErrInvalidTimeframe     = apperror.BadRequest("starts_at must be before or equal to ends_at")
)
