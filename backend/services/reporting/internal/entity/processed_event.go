package entity

import "time"

// ProcessedEvent records Kafka event IDs that have been successfully handled.
// Unique primary key on event_id provides the idempotency guard for all consumers.
type ProcessedEvent struct {
	EventID   string    `gorm:"type:varchar(80);primaryKey;column:event_id"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();column:created_at"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
