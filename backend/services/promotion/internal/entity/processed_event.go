package entity

import "time"

// ProcessedEvent records Kafka event IDs that have already been handled.
// Consumers insert the envelope event_id in the same transaction as their
// side effects; a duplicate delivery hits the PK and is skipped.
type ProcessedEvent struct {
	EventID     string    `gorm:"type:uuid;primaryKey"`
	ProcessedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
