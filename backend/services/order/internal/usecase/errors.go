package usecase

import (
	"net/http"

	"project/pkg/apperror"
)

// Domain errors returned by usecases and mapped to HTTP status codes.
var (
	ErrCartEmpty            = apperror.New(http.StatusBadRequest, "cart is empty")
	ErrStoreNotFound        = apperror.New(http.StatusBadRequest, "store not found")
	ErrStoreClosed          = apperror.New(http.StatusBadRequest, "cửa hàng đang đóng")
	ErrStoreNotServed       = apperror.New(http.StatusBadRequest, "delivery not available for your location")
	ErrItemNotInStore       = apperror.New(http.StatusBadRequest, "one or more items are not available in this store")
	ErrDesiredTimePast      = apperror.New(http.StatusBadRequest, "desired_time must be at least now (store prep time applies)")
	ErrDesiredTimeNotToday  = apperror.New(http.StatusBadRequest, "desired_time must be today")
	ErrDesiredTimeAfterClose = apperror.New(http.StatusBadRequest, "desired_time is after store closing time today")
	ErrPromotionInvalid     = apperror.New(http.StatusBadRequest, "promotion code is invalid or expired")
	ErrPaymentFailed        = apperror.New(http.StatusBadRequest, "thanh toán thất bại — vui lòng thử lại hoặc chọn phương thức khác")
	ErrGrandTotalNegative   = apperror.New(http.StatusBadRequest, "grand total must be ≥ 0 after discounts")
	ErrOrderNotFound        = apperror.New(http.StatusNotFound, "order not found")
	ErrCancelNotAllowed     = apperror.New(http.StatusBadRequest, "order can only be cancelled before it is confirmed")
	ErrInvalidTransition    = apperror.New(http.StatusBadRequest, "status transition is not allowed")
	ErrNotOrderOwner        = apperror.New(http.StatusForbidden, "you do not own this order")
	ErrPickupPINMismatch    = apperror.New(http.StatusBadRequest, "pickup PIN does not match")
)
