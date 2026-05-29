package entity

import "time"

// FacultyProfile extends a user with staff-specific data.
// UserID is both the PK and FK — one-to-one with users.
type FacultyProfile struct {
	UserID                string     `gorm:"type:uuid;primaryKey"`
	StaffCode             string     `gorm:"type:varchar(50);uniqueIndex;not null"`
	Department            *string    `gorm:"type:varchar(100)"`
	Position              *string    `gorm:"type:varchar(100)"`
	AllowPayrollDeduction bool       `gorm:"not null;default:false"`
	SyncedAt              *time.Time `gorm:"type:timestamptz"`

	User *User `gorm:"foreignKey:UserID"`
}

func (FacultyProfile) TableName() string {
	return "faculty_profiles"
}
