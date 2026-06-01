package usecase

import "project/pkg/apperror"

var (
	ErrOrderNotFound       = apperror.New(404, "order not found")
	ErrOrderNotCompleted   = apperror.New(400, "order is not completed")
	ErrWrongCustomer       = apperror.New(403, "order does not belong to this customer")
	ErrDuplicateReview     = apperror.New(409, "review already exists for this order and target")
	ErrReviewNotFound      = apperror.New(404, "review not found")
	ErrInvalidRating       = apperror.New(400, "rating must be between 1 and 5")
	ErrInvalidTargetType   = apperror.New(400, "target_type must be STORE, ITEM, or SHIPPER")
	ErrUnauthorizedReply   = apperror.New(403, "not authorised to reply to this review")
	ErrStoreNotFound       = apperror.New(404, "store not found")
)
