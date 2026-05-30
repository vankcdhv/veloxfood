package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/pkg/outbox"
	"project/services/store/internal/usecase"

	"github.com/segmentio/kafka-go"
)

// VendorEventHandler consumes events from the vendor.events Kafka topic.
// It reacts to vendor lifecycle events to create/close the corresponding store.
type VendorEventHandler struct {
	storeUC usecase.StoreUsecase
	quotaUC usecase.QuotaUsecase
}

func NewVendorEventHandler(storeUC usecase.StoreUsecase, quotaUC usecase.QuotaUsecase) *VendorEventHandler {
	return &VendorEventHandler{storeUC: storeUC, quotaUC: quotaUC}
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
	case "order.placed":
		return h.handleOrderPlaced(ctx, env)
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
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

// ─── Order events (stubs — Order service topic absent in current sprint) ──────

type orderPlacedData struct {
	StoreID    string `json:"store_id"`
	MenuItemID string `json:"menu_item_id"`
	CutoffID   string `json:"cutoff_id"`
	Date       string `json:"date"` // YYYY-MM-DD
	Qty        int    `json:"qty"`
}

// handleOrderPlaced decrements the slot quota for each ordered item.
// Wired but the order.events topic has no producer yet — handler is correct
// and will activate automatically once the Order service starts publishing.
func (h *VendorEventHandler) handleOrderPlaced(ctx context.Context, env outbox.Envelope) error {
	var data orderPlacedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order.placed: unmarshal failed", "err", err)
		return nil
	}

	date, err := time.Parse("2006-01-02", data.Date)
	if err != nil {
		slog.ErrorContext(ctx, "order.placed: invalid date", "date", data.Date)
		return nil
	}

	if err := h.quotaUC.DecrementSlot(ctx, data.MenuItemID, date, data.CutoffID, data.Qty); err != nil {
		slog.ErrorContext(ctx, "order.placed: decrement slot failed",
			"menu_item_id", data.MenuItemID, "err", err)
		return err
	}
	return nil
}

type orderCancelledData struct {
	MenuItemID string `json:"menu_item_id"`
	CutoffID   string `json:"cutoff_id"`
	Date       string `json:"date"`
	Qty        int    `json:"qty"`
}

// handleOrderCancelled restores the slot quota for a cancelled order.
func (h *VendorEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	var data orderCancelledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order.cancelled: unmarshal failed", "err", err)
		return nil
	}

	date, err := time.Parse("2006-01-02", data.Date)
	if err != nil {
		slog.ErrorContext(ctx, "order.cancelled: invalid date", "date", data.Date)
		return nil
	}

	if err := h.quotaUC.RestoreSlot(ctx, data.MenuItemID, date, data.CutoffID, data.Qty); err != nil {
		slog.ErrorContext(ctx, "order.cancelled: restore slot failed",
			"menu_item_id", data.MenuItemID, "err", err)
		return err
	}
	return nil
}
