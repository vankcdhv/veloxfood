package entity

import "time"

// Permission represents a single action on a resource (e.g. resource=user, action=suspend).
// code is the canonical identifier used in authorization checks (e.g. "user.suspend").
type Permission struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string    `gorm:"type:varchar(80);uniqueIndex;not null"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description *string   `gorm:"type:text"`
	Resource    string    `gorm:"type:varchar(40);not null"`
	Action      string    `gorm:"type:varchar(40);not null"`
	IsSystem    bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (Permission) TableName() string {
	return "permissions"
}
