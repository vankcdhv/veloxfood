package entity

import (
	"encoding/json"
	"time"
)

// AuditLog records privileged actions for compliance and debugging.
// payload JSONB must be redacted by the usecase layer before insertion
// — never include password_hash, raw tokens, or OTP codes in payload.
type AuditLog struct {
	ID          string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorUserID *string          `gorm:"type:uuid;index"`
	Action      string           `gorm:"type:varchar(80);not null"`
	TargetType  *string          `gorm:"type:varchar(40)"`
	TargetID    *string          `gorm:"type:uuid"`
	IP          *string          `gorm:"type:varchar(45)"`
	UserAgent   *string          `gorm:"type:text"`
	TraceID     *string          `gorm:"type:varchar(80)"`
	Payload     *json.RawMessage `gorm:"type:jsonb"`
	CreatedAt   time.Time        `gorm:"type:timestamptz;not null;default:now()"`

	Actor *User `gorm:"foreignKey:ActorUserID"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
