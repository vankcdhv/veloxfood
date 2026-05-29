package entity

import "time"

// PasswordResetToken is a single-use token for resetting a user's password.
// token_hash stores SHA-256 of the raw token — NEVER store the raw token.
// used_at being non-nil means the token has been consumed.
type PasswordResetToken struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string     `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:varchar(128);uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null"`
	UsedAt    *time.Time `gorm:"type:timestamptz"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`

	User *User `gorm:"foreignKey:UserID"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
