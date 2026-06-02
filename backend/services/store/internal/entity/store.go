package entity

import (
	"time"

	"gorm.io/gorm"
)

// Store is the top-level store aggregate (cửa hàng).
// SaleStatus values: OPEN | CLOSED_TODAY | PAUSED.
// VendorID links to user-service vendor_memberships.vendor_id (one store per vendor).
type Store struct {
	ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerUserID   string         `gorm:"type:uuid;not null;index"`
	VendorID      string         `gorm:"type:uuid;not null;uniqueIndex:uq_stores_vendor_active,where:deleted_at IS NULL"`
	Name          string         `gorm:"type:varchar(150);not null"`
	BusinessType  string         `gorm:"type:varchar(100)"`
	Address       string         `gorm:"type:varchar(255)"`
	Phone         string         `gorm:"type:varchar(30)"`
	SaleStatus    string         `gorm:"type:varchar(20);not null;default:'OPEN'"`
	PickupEnabled bool           `gorm:"not null;default:false"`
	PrepMinutes   int            `gorm:"not null;default:15"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt     gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Store) TableName() string { return "stores" }
