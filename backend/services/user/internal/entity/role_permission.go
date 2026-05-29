package entity

import "time"

// RolePermission is the join table between roles and permissions.
// Composite primary key: (role_id, permission_id).
type RolePermission struct {
	RoleID       string    `gorm:"type:uuid;not null;primaryKey"`
	PermissionID string    `gorm:"type:uuid;not null;primaryKey"`
	GrantedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
	// GrantedBy is a soft reference to users.id — no FK to avoid circular dep before users table.
	GrantedBy *string `gorm:"type:uuid"`

	// Associations (optional eager-load)
	Role       *Role       `gorm:"foreignKey:RoleID"`
	Permission *Permission `gorm:"foreignKey:PermissionID"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
