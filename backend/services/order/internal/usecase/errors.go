package usecase

import (
	"net/http"

	"project/pkg/apperror"
)

// Domain errors returned by usecases and mapped to HTTP status codes.
var (
	ErrCartCrossStore      = apperror.New(http.StatusBadRequest, "cart already belongs to a different store; clear it first")
	ErrCartEmpty           = apperror.New(http.StatusBadRequest, "cart is empty")
	ErrStoreNotFound       = apperror.New(http.StatusBadRequest, "store not found")
	ErrStoreClosed         = apperror.New(http.StatusBadRequest, "store is not accepting orders")
	ErrStoreNotServed      = apperror.New(http.StatusBadRequest, "delivery not available for your location")
	ErrOrderDeadlinePassed = apperror.New(http.StatusBadRequest, "order deadline has passed for this store")
	ErrPreOrderTooFar      = apperror.New(http.StatusBadRequest, "pre-order date must be within 7 days")
	ErrItemNotInStore      = apperror.New(http.StatusBadRequest, "one or more items are not available in this store")
	ErrQuotaExceeded       = apperror.New(http.StatusConflict, "slot quota exceeded for one or more items")
	ErrPromotionInvalid    = apperror.New(http.StatusBadRequest, "promotion code is invalid or expired")
	ErrGrandTotalNegative  = apperror.New(http.StatusBadRequest, "grand total must be ≥ 0 after discounts")
	ErrOrderNotFound       = apperror.New(http.StatusNotFound, "order not found")
	ErrCancelNotAllowed    = apperror.New(http.StatusBadRequest, "order can only be cancelled before it is confirmed")
	ErrInvalidTransition   = apperror.New(http.StatusBadRequest, "status transition is not allowed")
	ErrNotOrderOwner       = apperror.New(http.StatusForbidden, "you do not own this order")
	ErrPickupPINMismatch   = apperror.New(http.StatusBadRequest, "pickup PIN does not match")
)
