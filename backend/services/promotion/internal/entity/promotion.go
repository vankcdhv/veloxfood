package entity

import (
	"time"

	"gorm.io/gorm"
)

// Promotion represents a discount voucher scoped to a single store.
// type values: ORDER_DISCOUNT | SHIP_DISCOUNT
// value_kind values: PERCENT | AMOUNT
// status values: ACTIVE | INACTIVE
//
// No json tags — struct serialises PascalCase so HTTP responses match frontend
// expectations without a DTO translation layer.
type Promotion struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID     string         `gorm:"type:uuid;not null;index"`
	Code        string         `gorm:"type:varchar(50);not null"`
	Type        string         `gorm:"type:varchar(20);not null"`
	ValueKind   string         `gorm:"type:varchar(10);not null"`
	Value       int64          `gorm:"not null"`
	MinOrder    int64          `gorm:"not null;default:0"`
	MaxDiscount *int64         `gorm:"default:null"`
	StartsAt    time.Time      `gorm:"type:timestamptz;not null"`
	EndsAt      time.Time      `gorm:"type:timestamptz;not null"`
	UsageLimit  *int           `gorm:"default:null"`
	UsedCount   int            `gorm:"not null;default:0"`
	Status      string         `gorm:"type:varchar(10);not null;default:'ACTIVE'"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Promotion) TableName() string { return "promotions" }

// PromotionUsage tracks the reservation/confirmation/void lifecycle of a single
// promotion application to an order.
// status values: RESERVED | CONFIRMED | VOIDED
type PromotionUsage struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PromotionID   string    `gorm:"type:uuid;not null;index"`
	OrderID       string    `gorm:"type:uuid;not null;index"`
	CustomerID    string    `gorm:"type:uuid;not null"`
	Code          string    `gorm:"type:varchar(50);not null"`
	Type          string    `gorm:"type:varchar(20);not null"`
	AppliedAmount int64     `gorm:"not null"`
	Status        string    `gorm:"type:varchar(10);not null;default:'RESERVED'"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt     time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (PromotionUsage) TableName() string { return "promotion_usages" }
