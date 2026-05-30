package entity

import "time"

// User is the central identity record. password_hash stores bcrypt — never plaintext.
type User struct {
	ID                  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email               *string    `gorm:"type:varchar(255);uniqueIndex"`
	Phone               *string    `gorm:"type:varchar(50);uniqueIndex"`
	PasswordHash        *string    `gorm:"type:varchar(255)"`
	Status              UserStatus `gorm:"type:varchar(20);not null;default:'pending';index"`
	FullName            string     `gorm:"type:varchar(255);not null"`
	DOB                 *time.Time `gorm:"type:date"`
	Gender              *Gender    `gorm:"type:varchar(10)"`
	AvatarURL           *string    `gorm:"type:text"`
	FailedLoginAttempts int        `gorm:"not null;default:0"`
	LockedUntil         *time.Time `gorm:"type:timestamptz"`
	CreatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (User) TableName() string {
	return "users"
}
