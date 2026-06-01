package entity

import "time"

// OrderFact is a denormalised read-model projection of an order, built from
// consumed Kafka events. It is the primary source for analytics queries.
type OrderFact struct {
	OrderID       string    `gorm:"type:uuid;primaryKey;column:order_id"`
	Code          string    `gorm:"type:varchar(20);column:code"`
	StoreID       string    `gorm:"type:uuid;not null;index;column:store_id"`
	CustomerID    string    `gorm:"type:uuid;not null;column:customer_id"`
	Date          time.Time `gorm:"type:date;not null;column:date"`
	Fulfillment   string    `gorm:"type:varchar(10);not null;column:fulfillment"`
	ItemsTotal    int64     `gorm:"not null;default:0;column:items_total"`
	ShipFee       int64     `gorm:"not null;default:0;column:ship_fee"`
	Discount      int64     `gorm:"not null;default:0;column:discount"`
	GrandTotal    int64     `gorm:"not null;default:0;column:grand_total"`
	PaymentMethod string    `gorm:"type:varchar(10);not null;column:payment_method"`
	Status        string    `gorm:"type:varchar(20);not null;default:'PENDING';column:status"`
	Settled       bool      `gorm:"not null;default:false;column:settled"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:now();column:created_at"`
	UpdatedAt     time.Time `gorm:"type:timestamptz;not null;default:now();column:updated_at"`
}

func (OrderFact) TableName() string { return "order_facts" }
