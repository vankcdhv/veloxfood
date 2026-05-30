package entity

import (
	"time"

	"gorm.io/gorm"
)

// Building is the top of the delivery-address tree (Toà nhà).
type Building struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(150);not null"`
	Address   string         `gorm:"type:varchar(255)"`
	IsActive  bool           `gorm:"not null;default:true"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Building) TableName() string { return "buildings" }

// Floor belongs to a Building (Tầng).
type Floor struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BuildingID string         `gorm:"type:uuid;not null;index"`
	Name       string         `gorm:"type:varchar(100);not null"`
	SortOrder  int            `gorm:"not null;default:0"`
	CreatedAt  time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt  time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt  gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Floor) TableName() string { return "floors" }

// Room belongs to a Floor (Phòng). rooms.id is the system-wide location_id.
type Room struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FloorID   string         `gorm:"type:uuid;not null;index"`
	Code      string         `gorm:"type:varchar(50);not null"`
	Name      string         `gorm:"type:varchar(100)"`
	IsActive  bool           `gorm:"not null;default:true"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Room) TableName() string { return "rooms" }

// CustomerLocation is a customer's saved/favorite delivery location.
type CustomerLocation struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID string    `gorm:"type:uuid;not null;index"`
	RoomID     string    `gorm:"type:uuid;not null"`
	Label      string    `gorm:"type:varchar(100)"`
	IsDefault  bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (CustomerLocation) TableName() string { return "customer_locations" }
