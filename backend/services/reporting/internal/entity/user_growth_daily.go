package entity

import "time"

// UserGrowthDaily tracks new customer and vendor registrations per calendar day.
type UserGrowthDaily struct {
	Date         time.Time `gorm:"type:date;not null;primaryKey;column:date"`
	NewCustomers int       `gorm:"not null;default:0;column:new_customers"`
	NewVendors   int       `gorm:"not null;default:0;column:new_vendors"`
}

func (UserGrowthDaily) TableName() string { return "user_growth_daily" }
