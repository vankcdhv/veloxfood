package entity

import (
	"encoding/json"
	"time"
)

// OutboxEvent implements the transactional outbox pattern for reliable
// event publishing to Kafka/RabbitMQ. A background publisher reads
// rows with status='pending' and marks them 'published' after delivery.
// last_error stores the most recent failure reason for observability.
type OutboxEvent struct {
	ID            string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AggregateType string           `gorm:"type:varchar(40);not null"`
	AggregateID   string           `gorm:"type:uuid;not null"`
	EventType     string           `gorm:"type:varchar(80);not null"`
	Payload       json.RawMessage  `gorm:"type:jsonb;not null"`
	TraceID       *string          `gorm:"type:varchar(80)"`
	Status        OutboxStatus     `gorm:"type:varchar(20);not null;default:'pending';index"`
	Attempts      int              `gorm:"not null;default:0"`
	LastError     *string          `gorm:"type:text"`
	CreatedAt     time.Time        `gorm:"type:timestamptz;not null;default:now()"`
	PublishedAt   *time.Time       `gorm:"type:timestamptz"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
