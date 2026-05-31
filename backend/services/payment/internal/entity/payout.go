package entity

import (
	"encoding/json"
	"time"
)

// PayoutStatus tracks settlement lifecycle.
type PayoutStatus string

const (
	PayoutPending  PayoutStatus = "PENDING"
	PayoutSettled  PayoutStatus = "SETTLED"
)

// PayoutBatch is a store settlement batch created by an admin.
// Executing the batch debits STORE_PAYABLE and marks it SETTLED in one transaction.
type PayoutBatch struct {
	ID          string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID     string          `gorm:"type:uuid;not null"`
	PeriodFrom  time.Time       `gorm:"type:date;not null"`
	PeriodTo    time.Time       `gorm:"type:date;not null"`
	OrderIDs    json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	TotalAmount int64           `gorm:"not null;default:0"`
	Status      PayoutStatus    `gorm:"type:varchar(20);not null;default:'PENDING'"`
	SettledBy   *string         `gorm:"type:uuid"`
	SettledAt   *time.Time      `gorm:"type:timestamptz"`
	CreatedAt   time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (PayoutBatch) TableName() string { return "payout_batches" }
