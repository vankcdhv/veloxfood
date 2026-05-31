package entity

import "time"

// ProcessedEvent deduplicates incoming Kafka events.
// Consumers insert (event_id) before acting; duplicate means already handled.
type ProcessedEvent struct {
	EventID   string    `gorm:"type:varchar(80);primaryKey"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
