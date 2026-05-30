package entity

import "time"

// VendorMembership links a user to a vendor with a role and lifecycle status.
// vendor_id is a soft-reference (no FK) — vendor table lives in a separate service.
type VendorMembership struct {
	ID           string                 `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       string                 `gorm:"type:uuid;not null;index"`
	VendorID     string                 `gorm:"type:uuid;not null;index"`
	RoleInVendor RoleInVendor           `gorm:"type:varchar(20);not null"`
	Status       VendorMembershipStatus `gorm:"type:varchar(20);not null;default:'invited'"`
	InvitedBy    *string                `gorm:"type:uuid"`
	InvitedAt    time.Time              `gorm:"type:timestamptz;not null;default:now()"`
	JoinedAt     *time.Time             `gorm:"type:timestamptz"`

	User *User `gorm:"foreignKey:UserID"`
}

func (VendorMembership) TableName() string {
	return "vendor_memberships"
}
