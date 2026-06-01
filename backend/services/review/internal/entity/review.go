package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// StringSlice is a JSON-serialisable []string backed by jsonb.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(src any) error {
	var b []byte
	switch v := src.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return fmt.Errorf("StringSlice.Scan: unsupported type %T", src)
	}
	return json.Unmarshal(b, s)
}

// Review is the core aggregate. target_type: STORE | ITEM | SHIPPER.
// status: VISIBLE | HIDDEN.
// No json tags — serialises PascalCase so HTTP responses match FE expectations.
type Review struct {
	ID         string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID    string      `gorm:"type:uuid;not null"`
	CustomerID string      `gorm:"type:uuid;not null"`
	StoreID    string      `gorm:"type:uuid;not null;index"`
	TargetType string      `gorm:"type:varchar(10);not null"`
	TargetID   string      `gorm:"type:uuid;not null"`
	Rating     int         `gorm:"type:smallint;not null"`
	Comment    string      `gorm:"type:text;not null;default:''"`
	PhotoURLs  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Status     string      `gorm:"type:varchar(10);not null;default:'VISIBLE'"`
	CreatedAt  time.Time   `gorm:"type:timestamptz;not null;default:now()"`
}

func (Review) TableName() string { return "reviews" }

// ReviewReply is a store-owner response to a review.
type ReviewReply struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReviewID     string    `gorm:"type:uuid;not null;index"`
	AuthorUserID string    `gorm:"type:uuid;not null"`
	Content      string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ReviewReply) TableName() string { return "review_replies" }

// ReviewReport is a user-submitted content complaint.
// status: OPEN | RESOLVED.
type ReviewReport struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReviewID   string    `gorm:"type:uuid;not null;index"`
	ReportedBy string    `gorm:"type:uuid;not null"`
	Reason     string    `gorm:"type:text;not null"`
	Status     string    `gorm:"type:varchar(10);not null;default:'OPEN'"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ReviewReport) TableName() string { return "review_reports" }

// OutboxEvent mirrors the shared outbox schema for the review service.
type OutboxEvent struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AggregateType string    `gorm:"type:varchar(40);not null"`
	AggregateID   string    `gorm:"type:uuid;not null"`
	EventType     string    `gorm:"type:varchar(80);not null"`
	Payload       []byte    `gorm:"type:jsonb;not null"`
	TraceID       string    `gorm:"type:varchar(80)"`
	Status        string    `gorm:"type:varchar(20);not null;default:'pending'"`
	Attempts      int       `gorm:"not null;default:0"`
	LastError     string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:now()"`
	PublishedAt   *time.Time
}

func (OutboxEvent) TableName() string { return "outbox_events" }

// ProcessedEvent deduplicates incoming Kafka events.
type ProcessedEvent struct {
	EventID   string    `gorm:"type:varchar(80);primaryKey"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
