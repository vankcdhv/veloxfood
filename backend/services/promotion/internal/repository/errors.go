package repository

import "errors"

// ErrPromotionNotFound is returned when a promotion row cannot be located.
var ErrPromotionNotFound = errors.New("promotion not found")

// ErrUsageNotFound is returned when no usage row matches the given criteria.
var ErrUsageNotFound = errors.New("promotion usage not found")
