package entity

import "time"

// CardIdentifier links an RFID card or biometric face identifier to a user.
// A partial unique index on (kind, identifier) WHERE revoked_at IS NULL
// enforces that only one active identifier exists per card/face value.
// identifier stores the raw card ID or face embedding reference — consider
// encryption at rest for face data in a production deployment.
type CardIdentifier struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string     `gorm:"type:uuid;not null;index"`
	Kind        CardKind   `gorm:"type:varchar(10);not null"`
	Identifier  string     `gorm:"type:text;not null"`
	IssuedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	RevokedAt   *time.Time `gorm:"type:timestamptz"`
	LastUsedAt  *time.Time `gorm:"type:timestamptz"`

	User *User `gorm:"foreignKey:UserID"`
}

func (CardIdentifier) TableName() string {
	return "card_identifiers"
}
