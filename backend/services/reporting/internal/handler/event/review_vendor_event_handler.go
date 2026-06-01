package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/pkg/outbox"
	"project/services/reporting/internal/repository"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ReviewVendorEventHandler consumes review.events and vendor.events topics.
// review.created updates store_performance rating aggregates.
// vendor.approved increments user_growth_daily.new_vendors.
type ReviewVendorEventHandler struct {
	db            *gorm.DB
	projRepo      repository.ProjectionRepository
	processedRepo repository.ProcessedEventRepository
}

func NewReviewVendorEventHandler(
	db *gorm.DB,
	projRepo repository.ProjectionRepository,
	processedRepo repository.ProcessedEventRepository,
) *ReviewVendorEventHandler {
	return &ReviewVendorEventHandler{db: db, projRepo: projRepo, processedRepo: processedRepo}
}

// HandleReviewKafkaMessage routes messages from review.events.
func (h *ReviewVendorEventHandler) HandleReviewKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "reporting: review event unmarshal failed", "err", err)
		return nil
	}
	if env.EventType == "review.created" {
		return h.handleReviewCreated(ctx, env)
	}
	slog.DebugContext(ctx, "reporting: review event ignored", "event_type", env.EventType)
	return nil
}

// HandleVendorKafkaMessage routes messages from vendor.events.
func (h *ReviewVendorEventHandler) HandleVendorKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "reporting: vendor event unmarshal failed", "err", err)
		return nil
	}
	if env.EventType == "vendor.approved" {
		return h.handleVendorApproved(ctx, env)
	}
	slog.DebugContext(ctx, "reporting: vendor event ignored", "event_type", env.EventType)
	return nil
}

// reviewCreatedData matches the review.created payload from domain-events §2.
type reviewCreatedData struct {
	ReviewID string `json:"review_id"`
	StoreID  string `json:"store_id"`
	OrderID  string `json:"order_id"`
	Rating int `json:"rating"`
}

func (h *ReviewVendorEventHandler) handleReviewCreated(ctx context.Context, env outbox.Envelope) error {
	var data reviewCreatedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: review.created unmarshal failed", "err", err)
		return nil
	}
	if data.StoreID == "" || data.Rating < 1 || data.Rating > 5 {
		slog.WarnContext(ctx, "reporting: review.created invalid payload",
			"store_id", data.StoreID, "rating", data.Rating)
		return nil
	}
	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return h.projRepo.UpsertStorePerformance(ctx, tx, data.StoreID, int64(data.Rating))
	})
}

type vendorApprovedData struct {
	VendorID string `json:"vendor_id"`
}

func (h *ReviewVendorEventHandler) handleVendorApproved(ctx context.Context, env outbox.Envelope) error {
	var data vendorApprovedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "reporting: vendor.approved unmarshal failed", "err", err)
		return nil
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := h.processedRepo.MarkProcessed(ctx, tx, env.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return h.projRepo.IncrementUserGrowth(ctx, tx, today, 0, 1)
	})
}
