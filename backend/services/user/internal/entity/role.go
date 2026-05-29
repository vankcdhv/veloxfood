package entity

import "time"

// Role is a named collection of permissions. scope_type controls whether
// the role is global (e.g. SUPER_ADMIN) or vendor-scoped (e.g. VENDOR_OWNER).
type Role struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string    `gorm:"type:varchar(60);uniqueIndex;not null"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description *string   `gorm:"type:text"`
	ScopeType   ScopeType `gorm:"type:varchar(10);not null;default:'global'"`
	IsSystem    bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (Role) TableName() string {
	return "roles"
}
