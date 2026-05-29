package entity

import (
	"encoding/json"
	"time"
)

// Identity stores one SSO or local auth link per provider per user.
// raw_profile holds the provider's user-info payload — may contain PII;
// treat as sensitive and avoid logging.
type Identity struct {
	ID              string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          string           `gorm:"type:uuid;not null;index"`
	Provider        IdentityProvider `gorm:"type:varchar(20);not null"`
	ExternalSubject string           `gorm:"type:varchar(255);not null"`
	RawProfile      *json.RawMessage `gorm:"type:jsonb"`
	CreatedAt       time.Time        `gorm:"type:timestamptz;not null;default:now()"`

	User *User `gorm:"foreignKey:UserID"`
}

func (Identity) TableName() string {
	return "identities"
}
