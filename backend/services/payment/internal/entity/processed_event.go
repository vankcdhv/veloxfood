package entity

import "time"

// ProcessedEvent records Kafka event IDs that have already been handled.
// INSERT is skipped (duplicate key) when a consumer sees the same event_id twice,
// which provides exactly-once processing semantics within a single DB transaction.
type ProcessedEvent struct {
	EventID     string    `gorm:"type:uuid;primaryKey;column:event_id"`
	ProcessedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
