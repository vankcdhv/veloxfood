package entity

import (
	"encoding/json"
	"time"
)

// Cart holds items for a customer at a single store.
// One cart per (customer_id, store_id) — enforced by unique constraint.
type Cart struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID string     `gorm:"type:uuid;not null;index"`
	StoreID    string     `gorm:"type:uuid;not null"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	Items      []CartItem `gorm:"foreignKey:CartID"`
}

func (Cart) TableName() string { return "carts" }

// CartItem is a single menu item inside a cart with snapshot data filled in
// when the item is added and refreshed on quantity changes.
type CartItem struct {
	ID              string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CartID          string          `gorm:"type:uuid;not null;index;uniqueIndex:uq_cart_item"`
	MenuItemID      string          `gorm:"type:uuid;not null;uniqueIndex:uq_cart_item"`
	NameSnapshot    string          `gorm:"type:varchar(150);not null;default:''"`
	PriceSnapshot   int64           `gorm:"not null;default:0"`
	Qty             int             `gorm:"not null"`
	OptionsSnapshot json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (CartItem) TableName() string { return "cart_items" }
