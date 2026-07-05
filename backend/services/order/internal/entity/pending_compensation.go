package entity

import "time"

// CompensationAction identifies which saga rollback call must be re-driven.
type CompensationAction string

const (
	CompensationRefund       CompensationAction = "REFUND"
	CompensationReleaseUsage CompensationAction = "RELEASE_USAGE"
)

// PendingCompensation is a durable saga-rollback intent. Rows are written when
// the in-request compensation call fails and are retried by a background
// worker until the (idempotent, order-id-keyed) call succeeds.
type PendingCompensation struct {
	ID        string             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   string             `gorm:"type:uuid;not null"`
	Action    CompensationAction `gorm:"type:varchar(20);not null"`
	Amount    int64              `gorm:"not null;default:0"`
	Attempts  int                `gorm:"not null;default:0"`
	LastError *string
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DoneAt    *time.Time `gorm:"type:timestamptz"`
}

func (PendingCompensation) TableName() string { return "pending_compensations" }
