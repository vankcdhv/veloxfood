package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/notification/internal/entity"
	"project/services/notification/internal/usecase"

	"github.com/segmentio/kafka-go"
)

// VendorReviewEventHandler consumes vendor.events and review.events.
type VendorReviewEventHandler struct {
	uc *usecase.NotificationUsecase
}

func NewVendorReviewEventHandler(uc *usecase.NotificationUsecase) *VendorReviewEventHandler {
	return &VendorReviewEventHandler{uc: uc}
}

func (h *VendorReviewEventHandler) HandleVendorKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: vendor event unmarshal failed", "err", err)
		return nil
	}
	switch env.EventType {
	case "vendor.approved":
		return h.handleVendorApproved(ctx, env)
	case "vendor.rejected":
		return h.handleVendorRejected(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: vendor event unhandled", "event_type", env.EventType)
	}
	return nil
}

func (h *VendorReviewEventHandler) HandleReviewKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: review event unmarshal failed", "err", err)
		return nil
	}
	switch env.EventType {
	case "review.created":
		return h.handleReviewCreated(ctx, env)
	case "review.reported":
		return h.handleReviewReported(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: review event unhandled", "event_type", env.EventType)
	}
	return nil
}

// ── vendor.* ─────────────────────────────────────────────────────────────────

type vendorApprovedData struct {
	OwnerID   string `json:"owner_id"`
	VendorID  string `json:"vendor_id"`
	StoreName string `json:"store_name"`
}

type vendorRejectedData struct {
	OwnerID  string `json:"owner_id"`
	VendorID string `json:"vendor_id"`
	Reason   string `json:"reason"`
}

func (h *VendorReviewEventHandler) handleVendorApproved(ctx context.Context, env outbox.Envelope) error {
	var d vendorApprovedData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.OwnerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.OwnerID,
		Type:    "vendor.approved",
		Title:   "Cửa hàng đã được duyệt",
		Body:    "Chúc mừng! Cửa hàng của bạn đã được phê duyệt và hoạt động.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"vendor_id": d.VendorID, "store_name": d.StoreName},
	})
}

func (h *VendorReviewEventHandler) handleVendorRejected(ctx context.Context, env outbox.Envelope) error {
	var d vendorRejectedData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.OwnerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.OwnerID,
		Type:    "vendor.rejected",
		Title:   "Đăng ký cửa hàng bị từ chối",
		Body:    "Đăng ký cửa hàng của bạn bị từ chối. Lý do: " + d.Reason,
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"vendor_id": d.VendorID, "reason": d.Reason},
	})
}

// ── review.* ─────────────────────────────────────────────────────────────────

type reviewCreatedData struct {
	ReviewID   string  `json:"review_id"`
	StoreOwner string  `json:"store_owner_id"` // owner to notify
	StoreID    string  `json:"store_id"`
	CustomerID string  `json:"customer_id"`
	Rating     float64 `json:"rating"`
}

type reviewReportedData struct {
	ReviewID string `json:"review_id"`
	StoreID  string `json:"store_id"`
	Reason   string `json:"reason"`
}

func (h *VendorReviewEventHandler) handleReviewCreated(ctx context.Context, env outbox.Envelope) error {
	var d reviewCreatedData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil
	}
	// Notify store owner. If store_owner_id is not in payload, skip gracefully.
	if d.StoreOwner == "" {
		slog.DebugContext(ctx, "notification: review.created: store_owner_id absent, skipping")
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.StoreOwner,
		Type:    "review.created",
		Title:   "Đánh giá mới",
		Body:    "Cửa hàng của bạn vừa nhận được đánh giá mới.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"review_id": d.ReviewID, "store_id": d.StoreID, "rating": d.Rating},
	})
}

func (h *VendorReviewEventHandler) handleReviewReported(ctx context.Context, env outbox.Envelope) error {
	// review.reported is primarily for admin — no user-facing notification needed here.
	slog.DebugContext(ctx, "notification: review.reported received, no-op", "review_id", func() string {
		var d reviewReportedData
		_ = json.Unmarshal(env.Data, &d)
		return d.ReviewID
	}())
	return nil
}
