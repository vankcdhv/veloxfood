package usecase

import "errors"

var (
	ErrDeliveryNotFound    = errors.New("delivery not found")
	ErrAlreadyClaimed      = errors.New("delivery already claimed by another shipper")
	ErrNotDeliveryOwner    = errors.New("not the claiming shipper for this delivery")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrBatchFull           = errors.New("batch is full (max 5 active deliveries)")
)
