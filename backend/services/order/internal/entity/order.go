package entity

import (
	"encoding/json"
	"time"
)

// OrderStatus is the lifecycle state of an order.
type OrderStatus string

const (
	StatusPending         OrderStatus = "PENDING"
	StatusConfirmed       OrderStatus = "CONFIRMED"
	StatusPreparing       OrderStatus = "PREPARING"
	StatusReady           OrderStatus = "READY"
	StatusReadyPickup     OrderStatus = "READY_PICKUP"
	StatusShipperAssigned OrderStatus = "SHIPPER_ASSIGNED"
	StatusDelivering      OrderStatus = "DELIVERING"
	StatusDelivered       OrderStatus = "DELIVERED"
	StatusCompleted       OrderStatus = "COMPLETED"
	StatusCancelled       OrderStatus = "CANCELLED"
	StatusRejected        OrderStatus = "REJECTED"
)

// PaymentStatus tracks whether the order has been paid.
type PaymentStatus string

const (
	PaymentUnpaid PaymentStatus = "UNPAID"
	PaymentPaid   PaymentStatus = "PAID"
)

// Fulfillment mode for the order.
type Fulfillment string

const (
	FulfillmentDelivery Fulfillment = "DELIVERY"
	FulfillmentPickup   Fulfillment = "PICKUP"
)

// PaymentMethod for the order.
type PaymentMethod string

const (
	MethodCOD    PaymentMethod = "COD"
	MethodMoMo   PaymentMethod = "MOMO"
	MethodWallet PaymentMethod = "WALLET"
)

// Order is the core aggregate for a customer's purchase.
type Order struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code          string          `gorm:"type:varchar(20);not null;uniqueIndex"`
	CustomerID    string          `gorm:"type:uuid;not null;index"`
	StoreID       string          `gorm:"type:uuid;not null;index"`
	LocationID    *string         `gorm:"type:uuid"`
	Fulfillment   Fulfillment     `gorm:"type:varchar(10);not null"`
	Status        OrderStatus     `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	ItemsTotal    int64           `gorm:"not null;default:0"`
	ShipFee       int64           `gorm:"not null;default:0"`
	Discount      int64           `gorm:"not null;default:0"`
	GrandTotal    int64           `gorm:"not null;default:0"`
	PaymentMethod PaymentMethod   `gorm:"type:varchar(10);not null"`
	PaymentStatus PaymentStatus   `gorm:"type:varchar(10);not null;default:'UNPAID'"`
	VoucherCodes  json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	PickupPin     *string         `gorm:"type:varchar(10)"`
	PlacedAt      time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	CreatedAt     time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt     time.Time       `gorm:"type:timestamptz;not null;default:now()"`

	// Loaded via preload — not stored in this table.
	Items []OrderItem `gorm:"foreignKey:OrderID"`
}

func (Order) TableName() string { return "orders" }

// OrderItem is a line in an order with price/name snapshots taken at placement.
type OrderItem struct {
	ID              string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID         string          `gorm:"type:uuid;not null;index"`
	MenuItemID      string          `gorm:"type:uuid;not null"`
	NameSnapshot    string          `gorm:"type:varchar(150);not null"`
	PriceSnapshot   int64           `gorm:"not null"`
	Qty             int             `gorm:"not null"`
	CutoffID        *string         `gorm:"type:uuid"`
	Date            *time.Time      `gorm:"type:date"`
	OptionsSnapshot json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (OrderItem) TableName() string { return "order_items" }

// OrderStatusHistory records every status transition for audit and tracking.
type OrderStatusHistory struct {
	ID        string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   string      `gorm:"type:uuid;not null;index"`
	Status    OrderStatus `gorm:"type:varchar(20);not null"`
	ChangedBy *string     `gorm:"type:uuid"`
	Note      *string     `gorm:"type:varchar(255)"`
	CreatedAt time.Time   `gorm:"type:timestamptz;not null;default:now()"`
}

func (OrderStatusHistory) TableName() string { return "order_status_history" }

// ValidTransitions defines allowed next statuses from each current status.
// Enforced in the usecase layer before any DB write.
var ValidTransitions = map[OrderStatus][]OrderStatus{
	StatusPending:         {StatusConfirmed, StatusCancelled, StatusRejected},
	StatusConfirmed:       {StatusPreparing, StatusCancelled},
	StatusPreparing:       {StatusReady, StatusReadyPickup},
	StatusReady:           {StatusShipperAssigned, StatusDelivering, StatusDelivered},
	StatusReadyPickup:     {StatusCompleted},
	StatusShipperAssigned: {StatusDelivering},
	StatusDelivering:      {StatusDelivered},
	StatusDelivered:       {StatusCompleted},
	StatusCompleted:       {},
	StatusCancelled:       {},
	StatusRejected:        {},
}

// CanTransition reports whether transitioning from current to next is allowed.
func CanTransition(current, next OrderStatus) bool {
	for _, allowed := range ValidTransitions[current] {
		if allowed == next {
			return true
		}
	}
	return false
}
