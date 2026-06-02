package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"
	"project/services/store/internal/usecase"

	"github.com/segmentio/kafka-go"
)

// VendorEventHandler consumes events from the vendor.events Kafka topic.
// It reacts to vendor lifecycle events to create/close the corresponding store.
type VendorEventHandler struct {
	storeUC usecase.StoreUsecase
}

func NewVendorEventHandler(storeUC usecase.StoreUsecase) *VendorEventHandler {
	return &VendorEventHandler{storeUC: storeUC}
}

// HandleKafkaMessage routes an incoming Kafka message to the correct handler
// based on the event_type inside the outbox envelope.
func (h *VendorEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "vendor event received", "topic", msg.Topic, "key", string(msg.Key))

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "vendor event: unmarshal envelope failed", "err", err)
		// Non-retryable parse error — return nil so the consumer commits the offset.
		return nil
	}

	switch env.EventType {
	case "vendor.approved":
		return h.handleVendorApproved(ctx, env)
	case "vendor.rejected":
		return h.handleVendorRejected(ctx, env)
	default:
		slog.DebugContext(ctx, "vendor event: unhandled event type", "event_type", env.EventType)
	}
	return nil
}

// ─── Vendor events ────────────────────────────────────────────────────────────

type vendorApprovedData struct {
	VendorID string `json:"vendor_id"`
	AdminID  string `json:"admin_id"`
}

// handleVendorApproved creates a store for the newly approved vendor (idempotent).
func (h *VendorEventHandler) handleVendorApproved(ctx context.Context, env outbox.Envelope) error {
	var data vendorApprovedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "vendor.approved: unmarshal data failed", "err", err)
		return nil // skip malformed event
	}
	if data.VendorID == "" {
		slog.WarnContext(ctx, "vendor.approved: missing vendor_id")
		return nil
	}

	// CreateStore is idempotent — returns existing store if already created.
	store, err := h.storeUC.CreateStore(ctx, data.VendorID, "", data.VendorID+" store")
	if err != nil {
		slog.ErrorContext(ctx, "vendor.approved: create store failed", "vendor_id", data.VendorID, "err", err)
		return err // retryable
	}
	slog.InfoContext(ctx, "vendor.approved: store provisioned", "store_id", store.ID, "vendor_id", data.VendorID)
	return nil
}

type vendorRejectedData struct {
	VendorID string `json:"vendor_id"`
}

// handleVendorRejected closes (soft-deletes) the store for a rejected vendor.
func (h *VendorEventHandler) handleVendorRejected(ctx context.Context, env outbox.Envelope) error {
	var data vendorRejectedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "vendor.rejected: unmarshal data failed", "err", err)
		return nil
	}
	if data.VendorID == "" {
		return nil
	}

	store, err := h.storeUC.GetStoreByVendorID(ctx, data.VendorID)
	if err != nil {
		// Store may not exist yet (rejected before provisioned) — not an error.
		slog.DebugContext(ctx, "vendor.rejected: store not found, nothing to close", "vendor_id", data.VendorID)
		return nil
	}

	if err := h.storeUC.DeleteStore(ctx, store.ID); err != nil {
		slog.ErrorContext(ctx, "vendor.rejected: close store failed", "store_id", store.ID, "err", err)
		return err
	}
	slog.InfoContext(ctx, "vendor.rejected: store closed", "store_id", store.ID, "vendor_id", data.VendorID)
	return nil
}
