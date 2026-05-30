package entity

import "time"

// Invitation is a one-time token sent to email/phone to join a vendor.
// token_hash stores SHA-256 of the raw token — NEVER store the raw token.
// vendor_id is a soft-reference (vendor service owns vendor records).
type Invitation struct {
	ID               string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID         string           `gorm:"type:uuid;not null;index"`
	Email            *string          `gorm:"type:varchar(255)"`
	Phone            *string          `gorm:"type:varchar(50)"`
	RoleInVendor     RoleInVendor     `gorm:"type:varchar(20);not null"`
	TokenHash        string           `gorm:"type:varchar(128);uniqueIndex;not null"`
	ExpiresAt        time.Time        `gorm:"type:timestamptz;not null"`
	Status           InvitationStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	AcceptedByUserID *string          `gorm:"type:uuid"`
	CreatedAt        time.Time        `gorm:"type:timestamptz;not null;default:now()"`

	AcceptedByUser *User `gorm:"foreignKey:AcceptedByUserID"`
}

func (Invitation) TableName() string {
	return "invitations"
}
