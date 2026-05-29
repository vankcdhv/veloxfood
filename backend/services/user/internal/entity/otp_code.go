package entity

import "time"

// OTPCode stores a hashed one-time password for various verification flows.
// code_hash stores bcrypt or SHA-256 of the raw code — NEVER store raw code.
// destination is the delivery target: email address or phone number.
// user_id may be NULL for registration flows where the user does not yet exist.
type OTPCode struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      *string    `gorm:"type:uuid;index"`
	Purpose     OTPPurpose `gorm:"type:varchar(30);not null"`
	Destination string     `gorm:"type:varchar(255);not null"`
	CodeHash    string     `gorm:"type:varchar(128);not null"`
	ExpiresAt   time.Time  `gorm:"type:timestamptz;not null"`
	UsedAt      *time.Time `gorm:"type:timestamptz"`
	Attempts    int        `gorm:"not null;default:0"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`

	User *User `gorm:"foreignKey:UserID"`
}

func (OTPCode) TableName() string {
	return "otp_codes"
}
