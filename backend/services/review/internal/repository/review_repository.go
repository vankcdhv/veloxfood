package repository

import (
	"context"

	"project/services/review/internal/entity"

	"gorm.io/gorm"
)

// ReviewRepository is the pure contract for review persistence.
type ReviewRepository interface {
	Create(ctx context.Context, tx *gorm.DB, r *entity.Review) error
	GetByID(ctx context.Context, id string) (*entity.Review, error)
	GetByOrderAndTarget(ctx context.Context, orderID, targetType, targetID string) (*entity.Review, error)
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entity.Review, int64, error)
	UpdateStatus(ctx context.Context, id, status string) error
	RatingSummary(ctx context.Context, storeID string) (avg float64, count int32, err error)
}

// ReviewReplyRepository persists owner replies.
type ReviewReplyRepository interface {
	Create(ctx context.Context, reply *entity.ReviewReply) error
	GetByReviewID(ctx context.Context, reviewID string) (*entity.ReviewReply, error)
}

// ReviewReportRepository persists content reports.
type ReviewReportRepository interface {
	Create(ctx context.Context, report *entity.ReviewReport) error
	ListOpen(ctx context.Context, limit, offset int) ([]*entity.ReviewReport, int64, error)
	Resolve(ctx context.Context, reviewID string) error
}

// OutboxRepository stores outgoing domain events for relay to Kafka.
// Pickup/MarkPublished/MarkFailed are also implemented so the adapter can
// satisfy pkg/outbox.DispatchRepository without a separate type.
type OutboxRepository interface {
	Insert(ctx context.Context, tx *gorm.DB, evt *entity.OutboxEvent) error
	Pickup(ctx context.Context, limit, maxAttempts int) ([]*entity.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}

// ProcessedEventRepository deduplicates incoming Kafka events.
type ProcessedEventRepository interface {
	Insert(ctx context.Context, tx *gorm.DB, eventID string) error
}
