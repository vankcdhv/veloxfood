package repository

import "errors"

// ErrQuotaExceeded is returned by DecrementSlotQuota when the atomic
// WHERE guard (sold_count + qty <= quota) matches zero rows, meaning
// the requested quantity is unavailable for that slot.
var ErrQuotaExceeded = errors.New("store: slot quota exceeded")
