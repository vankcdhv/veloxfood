package entity

import "time"

// DeliveryStatus is the lifecycle state of a delivery record.
type DeliveryStatus string

const (
	DeliveryAvailable      DeliveryStatus = "AVAILABLE"
	DeliveryClaimed        DeliveryStatus = "CLAIMED"
	DeliveryPickedUp       DeliveryStatus = "PICKED_UP"
	DeliveryDelivering     DeliveryStatus = "DELIVERING"
	DeliveryDelivered      DeliveryStatus = "DELIVERED"
	DeliveryCancelled      DeliveryStatus = "CANCELLED"
	DeliveryStoreDelivering DeliveryStatus = "STORE_DELIVERING"
)

// Delivery tracks a single order's delivery lifecycle.
// order_id is UNIQUE — one delivery record per order.
type Delivery struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     string         `gorm:"type:uuid;not null;uniqueIndex"`
	StoreID     string         `gorm:"type:uuid;not null;index"`
	LocationID  string         `gorm:"type:uuid;not null"`
	CustomerID  string         `gorm:"type:uuid;not null"`
	ShipperID   *string        `gorm:"type:uuid;index"`
	ShipFee     int64          `gorm:"not null;default:0"`
	Status      DeliveryStatus `gorm:"type:varchar(20);not null;default:'AVAILABLE';index"`
	BatchID     *string        `gorm:"type:uuid"`
	ClaimedAt   *time.Time     `gorm:"type:timestamptz"`
	DeliveredAt *time.Time     `gorm:"type:timestamptz"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
}

func (Delivery) TableName() string { return "deliveries" }

// ValidDeliveryTransitions defines allowed shipper status transitions.
var ValidDeliveryTransitions = map[DeliveryStatus]DeliveryStatus{
	DeliveryClaimed:    DeliveryPickedUp,
	DeliveryPickedUp:   DeliveryDelivering,
	DeliveryDelivering: DeliveryDelivered,
}

// CanTransitionTo reports whether a shipper-driven transition from current to next is valid.
func (d *Delivery) CanTransitionTo(next DeliveryStatus) bool {
	allowed, ok := ValidDeliveryTransitions[d.Status]
	return ok && allowed == next
}
