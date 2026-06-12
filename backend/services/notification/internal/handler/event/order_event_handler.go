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

// OrderEventHandler consumes order.events and fans out notifications.
type OrderEventHandler struct {
	uc *usecase.NotificationUsecase
}

func NewOrderEventHandler(uc *usecase.NotificationUsecase) *OrderEventHandler {
	return &OrderEventHandler{uc: uc}
}

func (h *OrderEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: order event unmarshal failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "order.placed":
		return h.handleOrderPlaced(ctx, env)
	case "order.confirmed":
		return h.handleOrderConfirmed(ctx, env)
	case "order.cancelled":
		return h.handleOrderCancelled(ctx, env)
	case "order.rejected":
		return h.handleOrderRejected(ctx, env)
	case "order.delivered":
		return h.handleOrderDelivered(ctx, env)
	case "order.completed":
		return h.handleOrderCompleted(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: order event unhandled", "event_type", env.EventType)
	}
	return nil
}

// orderEventData covers the common fields present in order.* payloads (§2bis).
type orderEventData struct {
	OrderID       string `json:"order_id"`
	Code          string `json:"code"`
	CustomerID    string `json:"customer_id"`
	StoreID       string `json:"store_id"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method"`
	GrandTotal    int64  `json:"grand_total"`
	CancelledBy   string `json:"cancelled_by"`
}

func parseOrderData(env outbox.Envelope) (orderEventData, bool) {
	var d orderEventData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return d, false
	}
	return d, d.CustomerID != "" || d.OrderID != ""
}

func (h *OrderEventHandler) handleOrderPlaced(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.placed",
		Title:   "Đặt hàng thành công",
		Body:    "Đơn hàng #" + d.Code + " đã được đặt. Chờ quán xác nhận.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{
			"status":   "PENDING",
			"order_id": d.OrderID,
			"code":     d.Code,
		},
	})
}

func (h *OrderEventHandler) handleOrderConfirmed(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.confirmed",
		Title:   "Quán đã xác nhận",
		Body:    "Đơn hàng #" + d.Code + " đang được chuẩn bị.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{"status": "CONFIRMED"},
	})
}

func (h *OrderEventHandler) handleOrderCancelled(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	// A store-initiated cancel is a rejection from the customer's perspective —
	// reuse the order.cancelled event (so refund + quota restore still flow) but
	// phrase the notification accordingly.
	title, body := "Đơn hàng đã huỷ", "Đơn hàng #"+d.Code+" đã bị huỷ."
	if d.CancelledBy == "store" {
		title, body = "Đơn hàng bị từ chối", "Đơn hàng #"+d.Code+" đã bị quán từ chối."
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.cancelled",
		Title:   title,
		Body:    body,
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{"status": "CANCELLED"},
	})
}

func (h *OrderEventHandler) handleOrderRejected(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.rejected",
		Title:   "Đơn hàng bị từ chối",
		Body:    "Đơn hàng #" + d.Code + " đã bị quán từ chối.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{"status": "REJECTED"},
	})
}

func (h *OrderEventHandler) handleOrderDelivered(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.delivered",
		Title:   "Đơn hàng đã giao",
		Body:    "Đơn hàng #" + d.Code + " đã được giao thành công.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{"status": "DELIVERED"},
	})
}

func (h *OrderEventHandler) handleOrderCompleted(ctx context.Context, env outbox.Envelope) error {
	d, ok := parseOrderData(env)
	if !ok || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "order.completed",
		Title:   "Hoàn thành đơn hàng",
		Body:    "Đơn hàng #" + d.Code + " đã hoàn thành. Cảm ơn bạn!",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "code": d.Code},
		OrderID: d.OrderID,
		FSFields: map[string]any{"status": "COMPLETED"},
	})
}
