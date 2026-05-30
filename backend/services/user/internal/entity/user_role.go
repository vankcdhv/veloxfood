package entity

import "time"

// UserRole assigns a role to a user within an optional scope (global or vendor-scoped).
// role_id is a FK to roles.id (D7: replaced legacy role_code TEXT column).
// Partial unique indexes in DB enforce: one global role per user, one vendor-role per (user, vendor).
type UserRole struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string     `gorm:"type:uuid;not null;index"`
	RoleID    string     `gorm:"type:uuid;not null;index"`
	ScopeType ScopeType  `gorm:"type:varchar(10);not null"`
	ScopeID   *string    `gorm:"type:uuid"`
	GrantedBy *string    `gorm:"type:uuid"`
	GrantedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	ExpiresAt *time.Time `gorm:"type:timestamptz"`

	User *User `gorm:"foreignKey:UserID"`
	Role *Role `gorm:"foreignKey:RoleID"`
}

func (UserRole) TableName() string {
	return "user_roles"
}
