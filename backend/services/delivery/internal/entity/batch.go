package entity

import "time"

// BatchStatus is the state of a delivery batch.
type BatchStatus string

const (
	BatchOpen   BatchStatus = "OPEN"
	BatchClosed BatchStatus = "CLOSED"
)

// DeliveryBatch groups deliveries for a shipper at a single store.
// One OPEN batch per (shipper, store) at a time. Max 5 active deliveries per batch.
type DeliveryBatch struct {
	ID        string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ShipperID string      `gorm:"type:uuid;not null;index"`
	StoreID   string      `gorm:"type:uuid;not null"`
	MaxSize   int         `gorm:"not null;default:5"`
	Status    BatchStatus `gorm:"type:varchar(10);not null;default:'OPEN'"`
	CreatedAt time.Time   `gorm:"type:timestamptz;not null;default:now()"`
}

func (DeliveryBatch) TableName() string { return "delivery_batches" }
