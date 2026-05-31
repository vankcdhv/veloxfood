package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/outbox"

	"github.com/segmentio/kafka-go"
)

// StoreEventHandler consumes store.events for the order service.
// Currently handles store.status_changed: Order service caches store status
// so place-order can block immediately without calling Store gRPC.
//
// MVP: we call Store.GetStoreForOrder live on each placement so no local cache
// is maintained. This handler logs the event and is ready for an extension
// when a local cache (Redis) is added.
type StoreEventHandler struct{}

func NewStoreEventHandler() *StoreEventHandler { return &StoreEventHandler{} }

func (h *StoreEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "order: store event received", "topic", msg.Topic)

	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "order: store event: unmarshal envelope failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "store.status_changed":
		return h.handleStoreStatusChanged(ctx, env)
	default:
		slog.DebugContext(ctx, "order: store event: unhandled type", "event_type", env.EventType)
	}
	return nil
}

type storeStatusChangedData struct {
	StoreID string `json:"store_id"`
	Status  string `json:"status"` // OPEN | CLOSED_TODAY | PAUSED
}

func (h *StoreEventHandler) handleStoreStatusChanged(ctx context.Context, env outbox.Envelope) error {
	var data storeStatusChangedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		slog.ErrorContext(ctx, "order: store.status_changed: unmarshal failed", "err", err)
		return nil
	}
	// Live validation via Store gRPC on each order placement means no local state
	// needs to be updated here. Log for observability.
	slog.InfoContext(ctx, "order: store status changed",
		"store_id", data.StoreID, "status", data.Status)
	return nil
}
