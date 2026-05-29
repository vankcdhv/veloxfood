package entity

import (
	"encoding/json"
	"time"
)

// StudentProfile extends a user with school-specific data.
// UserID is both the PK and FK — one-to-one with users.
type StudentProfile struct {
	UserID        string           `gorm:"type:uuid;primaryKey"`
	StudentCode   string           `gorm:"type:varchar(50);uniqueIndex;not null"`
	Faculty       *string          `gorm:"type:varchar(100)"`
	Class         *string          `gorm:"type:varchar(50)"`
	CohortYear    *int             `gorm:"type:int"`
	Allergies     *json.RawMessage `gorm:"type:jsonb"`
	DormitoryRoom *string          `gorm:"type:varchar(50)"`
	SyncedAt      *time.Time       `gorm:"type:timestamptz"`

	User *User `gorm:"foreignKey:UserID"`
}

func (StudentProfile) TableName() string {
	return "student_profiles"
}
