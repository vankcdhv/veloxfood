package entity

import "time"

// IncidentStatus tracks resolution state of a delivery incident.
type IncidentStatus string

const (
	IncidentOpen     IncidentStatus = "OPEN"
	IncidentResolved IncidentStatus = "RESOLVED"
)

// DeliveryIncident records a problem reported by a shipper during delivery.
type DeliveryIncident struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DeliveryID string         `gorm:"type:uuid;not null;index"`
	OrderID    string         `gorm:"type:uuid;not null"`
	ShipperID  string         `gorm:"type:uuid;not null"`
	Type       string         `gorm:"type:varchar(50);not null"`
	Note       string         `gorm:"type:text;not null"`
	PhotoURL   *string        `gorm:"type:text"`
	Status     IncidentStatus `gorm:"type:varchar(10);not null;default:'OPEN'"`
	CreatedAt  time.Time      `gorm:"type:timestamptz;not null;default:now()"`
}

func (DeliveryIncident) TableName() string { return "delivery_incidents" }
