package entity

import (
	"time"

	"gorm.io/gorm"
)

// ShipFeeRule defines delivery fee for a store scoped to a building, floor, or room.
// Scope values: "building" | "floor" | "room".
// Resolve logic (cascade from most specific): room > floor > building.
// Building-scope rule is required before adding floor- or room-scope rules
// for the same ancestor building (enforced in ShipFeeUsecase.CreateShipFeeRule).
// unit_fee >= 0 is valid; 0 means free delivery to that location.
// No FK on ref_id — it references location-service entities across service boundary.
// Unique on (store_id, scope, ref_id).
type ShipFeeRule struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID   string         `gorm:"type:uuid;not null;index;uniqueIndex:uq_ship_fee_rule"`
	Scope     string         `gorm:"type:varchar(10);not null;uniqueIndex:uq_ship_fee_rule"`
	RefID     string         `gorm:"type:uuid;not null;uniqueIndex:uq_ship_fee_rule"`
	UnitFee   int64          `gorm:"not null"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (ShipFeeRule) TableName() string { return "ship_fee_rules" }

// OperatingHours stores per-weekday open/close times for a store.
// Weekday: 0=Sunday … 6=Saturday.
// OpenTime/CloseTime format: HH:MM (e.g. "08:00", "22:30").
type OperatingHours struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID   string         `gorm:"type:uuid;not null;index"`
	Weekday   int16          `gorm:"not null"`
	OpenTime  string         `gorm:"type:varchar(5);not null"`
	CloseTime string         `gorm:"type:varchar(5);not null"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (OperatingHours) TableName() string { return "operating_hours" }

// ShipCutoff defines a shipment cut-off time slot for a store.
// CutoffTime format: HH:MM. LeadMinutes: order deadline = cutoff − lead_minutes (BR8).
type ShipCutoff struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID     string         `gorm:"type:uuid;not null;index"`
	CutoffTime  string         `gorm:"type:varchar(5);not null"`
	LeadMinutes int            `gorm:"not null;default:0"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (ShipCutoff) TableName() string { return "ship_cutoffs" }

// OperatingHoursChangeRequest is a pending request to change hours/cutoffs
// that requires Admin approval before taking effect (§2.5).
// Status values: pending | approved | rejected.
// Payload is stored as JSONB; no gorm.io/datatypes dependency needed.
// ReviewedBy is nullable — only set once an admin acts on the request.
type OperatingHoursChangeRequest struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID    string    `gorm:"type:uuid;not null;index"`
	Payload    string    `gorm:"type:jsonb;not null"`
	Status     string    `gorm:"type:varchar(10);not null;default:'pending'"`
	ReviewedBy *string   `gorm:"type:uuid"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (OperatingHoursChangeRequest) TableName() string { return "operating_hours_change_requests" }
