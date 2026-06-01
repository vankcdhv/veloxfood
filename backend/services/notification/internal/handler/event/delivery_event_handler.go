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

// DeliveryEventHandler consumes delivery.events and fans out notifications.
type DeliveryEventHandler struct {
	uc *usecase.NotificationUsecase
}

func NewDeliveryEventHandler(uc *usecase.NotificationUsecase) *DeliveryEventHandler {
	return &DeliveryEventHandler{uc: uc}
}

func (h *DeliveryEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: delivery event unmarshal failed", "err", err)
		return nil
	}

	switch env.EventType {
	case "delivery.claimed":
		return h.handleDeliveryClaimed(ctx, env)
	case "delivery.status_changed":
		return h.handleDeliveryStatusChanged(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: delivery event unhandled", "event_type", env.EventType)
	}
	return nil
}

type deliveryClaimedData struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	ShipperID  string `json:"shipper_id"`
}

type deliveryStatusChangedData struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	ShipperID  string `json:"shipper_id"`
	Status     string `json:"status"`
}

func (h *DeliveryEventHandler) handleDeliveryClaimed(ctx context.Context, env outbox.Envelope) error {
	var d deliveryClaimedData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "delivery.claimed",
		Title:   "Shipper đã nhận đơn",
		Body:    "Tài xế đang đến lấy đơn hàng của bạn.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "shipper_id": d.ShipperID},
		OrderID: d.OrderID,
		FSFields: map[string]any{
			"status":     "SHIPPER_ASSIGNED",
			"shipper_id": d.ShipperID,
		},
	})
}

func (h *DeliveryEventHandler) handleDeliveryStatusChanged(ctx context.Context, env outbox.Envelope) error {
	var d deliveryStatusChangedData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.CustomerID == "" {
		return nil
	}

	var title, body, notifType string
	fsStatus := d.Status

	switch d.Status {
	case "DELIVERING":
		notifType = "delivery.delivering"
		title = "Shipper đang giao hàng"
		body = "Tài xế đang trên đường giao đơn hàng của bạn."
	case "DELIVERED":
		notifType = "delivery.delivered"
		title = "Đơn hàng đã giao"
		body = "Tài xế đã giao đơn hàng thành công."
	default:
		// PICKED_UP and other statuses — write Firestore but skip in-app.
		if d.OrderID != "" {
			fsFields := map[string]any{"status": fsStatus}
			if d.ShipperID != "" {
				fsFields["shipper_id"] = d.ShipperID
			}
			_ = h.uc.FanOut(ctx, usecase.FanOutRequest{
				UserID:   d.CustomerID,
				OrderID:  d.OrderID,
				FSFields: fsFields,
			})
		}
		return nil
	}

	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    notifType,
		Title:   title,
		Body:    body,
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "shipper_id": d.ShipperID},
		OrderID: d.OrderID,
		FSFields: map[string]any{
			"status":     fsStatus,
			"shipper_id": d.ShipperID,
		},
	})
}
