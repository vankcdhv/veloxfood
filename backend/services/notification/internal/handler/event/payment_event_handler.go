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

// PaymentEventHandler consumes payment.events, wallet.events, payout.events.
type PaymentEventHandler struct {
	uc *usecase.NotificationUsecase
}

func NewPaymentEventHandler(uc *usecase.NotificationUsecase) *PaymentEventHandler {
	return &PaymentEventHandler{uc: uc}
}

func (h *PaymentEventHandler) HandlePaymentKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: payment event unmarshal failed", "err", err)
		return nil
	}
	switch env.EventType {
	case "payment.captured":
		return h.handlePaymentCaptured(ctx, env)
	case "payment.failed":
		return h.handlePaymentFailed(ctx, env)
	case "payment.refunded":
		return h.handlePaymentRefunded(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: payment event unhandled", "event_type", env.EventType)
	}
	return nil
}

func (h *PaymentEventHandler) HandleWalletKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: wallet event unmarshal failed", "err", err)
		return nil
	}
	switch env.EventType {
	case "wallet.topped_up":
		return h.handleWalletToppedUp(ctx, env)
	case "wallet.debited":
		return h.handleWalletDebited(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: wallet event unhandled", "event_type", env.EventType)
	}
	return nil
}

func (h *PaymentEventHandler) HandlePayoutKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.ErrorContext(ctx, "notification: payout event unmarshal failed", "err", err)
		return nil
	}
	switch env.EventType {
	case "payout.settled":
		return h.handlePayoutSettled(ctx, env)
	default:
		slog.DebugContext(ctx, "notification: payout event unhandled", "event_type", env.EventType)
	}
	return nil
}

// ── payment.* ────────────────────────────────────────────────────────────────

type paymentData struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Amount     int64  `json:"amount"`
	Reason     string `json:"reason"`
}

func (h *PaymentEventHandler) handlePaymentCaptured(ctx context.Context, env outbox.Envelope) error {
	var d paymentData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "payment.captured",
		Title:   "Thanh toán thành công",
		Body:    "Đơn hàng đã được thanh toán thành công.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "amount": d.Amount},
	})
}

func (h *PaymentEventHandler) handlePaymentFailed(ctx context.Context, env outbox.Envelope) error {
	var d paymentData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "payment.failed",
		Title:   "Thanh toán thất bại",
		Body:    "Không thể xử lý thanh toán cho đơn hàng của bạn.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "reason": d.Reason},
	})
}

func (h *PaymentEventHandler) handlePaymentRefunded(ctx context.Context, env outbox.Envelope) error {
	var d paymentData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.CustomerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.CustomerID,
		Type:    "payment.refunded",
		Title:   "Hoàn tiền thành công",
		Body:    "Đơn hàng đã được hoàn tiền.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"order_id": d.OrderID, "amount": d.Amount},
	})
}

// ── wallet.* ─────────────────────────────────────────────────────────────────

type walletData struct {
	UserID  string `json:"user_id"`
	Amount  int64  `json:"amount"`
	Balance int64  `json:"balance"`
	OrderID string `json:"order_id"`
}

func (h *PaymentEventHandler) handleWalletToppedUp(ctx context.Context, env outbox.Envelope) error {
	var d walletData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.UserID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.UserID,
		Type:    "wallet.topped_up",
		Title:   "Nạp tiền thành công",
		Body:    "Ví của bạn đã được nạp tiền.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"amount": d.Amount, "balance": d.Balance},
	})
}

func (h *PaymentEventHandler) handleWalletDebited(ctx context.Context, env outbox.Envelope) error {
	var d walletData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.UserID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.UserID,
		Type:    "wallet.debited",
		Title:   "Thanh toán qua ví",
		Body:    "Ví của bạn đã bị trừ tiền cho đơn hàng.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"amount": d.Amount, "balance": d.Balance, "order_id": d.OrderID},
	})
}

// ── payout.* ─────────────────────────────────────────────────────────────────

type payoutData struct {
	OwnerID string `json:"owner_id"`
	StoreID string `json:"store_id"`
	Amount  int64  `json:"amount"`
}

func (h *PaymentEventHandler) handlePayoutSettled(ctx context.Context, env outbox.Envelope) error {
	var d payoutData
	if err := json.Unmarshal(env.Data, &d); err != nil || d.OwnerID == "" {
		return nil
	}
	return h.uc.FanOut(ctx, usecase.FanOutRequest{
		EventID: env.EventID,
		UserID:  d.OwnerID,
		Type:    "payout.settled",
		Title:   "Đã thanh toán doanh thu",
		Body:    "Doanh thu tháng của cửa hàng bạn đã được chuyển khoản.",
		Channel: entity.ChannelInApp,
		Data:    map[string]any{"store_id": d.StoreID, "amount": d.Amount},
	})
}
