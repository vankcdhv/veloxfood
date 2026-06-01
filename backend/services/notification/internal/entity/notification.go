package entity

import (
	"encoding/json"
	"time"
)

// NotificationChannel controls the delivery path for a notification.
type NotificationChannel string

const (
	ChannelPush   NotificationChannel = "push"
	ChannelEmail  NotificationChannel = "email"
	ChannelInApp  NotificationChannel = "in_app"
)

// Notification is an in-app / push / email record persisted per user.
type Notification struct {
	ID        string              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string              `gorm:"type:uuid;not null;index"`
	Type      string              `gorm:"type:varchar(60);not null"`
	Title     string              `gorm:"type:varchar(255);not null"`
	Body      string              `gorm:"type:text;not null"`
	Data      json.RawMessage     `gorm:"type:jsonb;not null;default:'{}'"`
	Channel   NotificationChannel `gorm:"type:varchar(20);not null;default:'in_app'"`
	ReadAt    *time.Time          `gorm:"type:timestamptz"`
	CreatedAt time.Time           `gorm:"type:timestamptz;not null;default:now()"`
}

func (Notification) TableName() string { return "notifications" }

// DeviceToken holds an FCM push token for a user's device.
type DeviceToken struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	FCMToken  string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Platform  string    `gorm:"type:varchar(20);not null;default:'web'"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (DeviceToken) TableName() string { return "device_tokens" }

// ProcessedEvent deduplicates incoming Kafka events.
type ProcessedEvent struct {
	EventID   string    `gorm:"type:varchar(80);primaryKey"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
