package usecase

import "errors"

var (
	// ErrInsufficientBalance is returned when a wallet debit would go below zero.
	ErrInsufficientBalance = errors.New("insufficient wallet balance")

	// ErrPaymentNotFound is returned when no payment exists for the given reference.
	ErrPaymentNotFound = errors.New("payment not found")

	// ErrAlreadyRefunded is returned when a refund is attempted on an already-refunded payment.
	ErrAlreadyRefunded = errors.New("payment already refunded")

	// ErrPayoutAlreadySettled is returned when an execute is attempted on a SETTLED batch.
	ErrPayoutAlreadySettled = errors.New("payout batch already settled")

	// ErrPayoutNotFound is returned when the batch does not exist.
	ErrPayoutNotFound = errors.New("payout batch not found")

	// ErrNothingToSettle is returned when a store has no settleable balance to pay out.
	ErrNothingToSettle = errors.New("nothing to settle for this store")
)
