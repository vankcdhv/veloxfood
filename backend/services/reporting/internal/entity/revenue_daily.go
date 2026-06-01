package entity

import "time"

// RevenueDaily aggregates order revenue and counts per store per day.
type RevenueDaily struct {
	StoreID        string    `gorm:"type:uuid;not null;primaryKey;column:store_id"`
	Date           time.Time `gorm:"type:date;not null;primaryKey;column:date"`
	TotalFood      int64     `gorm:"not null;default:0;column:total_food"`
	TotalShip      int64     `gorm:"not null;default:0;column:total_ship"`
	OrdersCount    int       `gorm:"not null;default:0;column:orders_count"`
	CancelledCount int       `gorm:"not null;default:0;column:cancelled_count"`
}

func (RevenueDaily) TableName() string { return "revenue_daily" }
