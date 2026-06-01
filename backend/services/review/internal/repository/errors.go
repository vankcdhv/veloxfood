package repository

import "errors"

var (
	ErrReviewNotFound  = errors.New("review not found")
	ErrDuplicateReview = errors.New("review already exists for this order/target")
)
