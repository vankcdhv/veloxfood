package entity

import "time"

// ShipperProfile holds the minimal shipper onboarding data.
// Name/phone live on the User; here we only keep the verification documents
// (stored in MinIO — DB holds object keys/URLs) and the Admin approval state.
type ShipperProfile struct {
	UserID             string        `gorm:"type:uuid;primaryKey"`
	IDDocumentPhotoURL string        `gorm:"type:text;not null"`
	PortraitPhotoURL   string        `gorm:"type:text;not null"`
	Status             ShipperStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	ApprovedBy         *string       `gorm:"type:uuid"`
	ApprovedAt         *time.Time    `gorm:"type:timestamptz"`
	CreatedAt          time.Time     `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt          time.Time     `gorm:"type:timestamptz;not null;default:now()"`

	User *User `gorm:"foreignKey:UserID"`
}

func (ShipperProfile) TableName() string {
	return "shipper_profiles"
}
