package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/firebase"
	"project/pkg/mailer"
	"project/services/notification/internal/entity"
	"project/services/notification/internal/repository"

	"gorm.io/gorm"
)

// NotificationUsecase handles in-app notification management and fan-out.
type NotificationUsecase struct {
	db           *gorm.DB
	notifRepo    repository.NotificationRepository
	tokenRepo    repository.DeviceTokenRepository
	processedRepo repository.ProcessedEventRepository
	firebase     *firebase.Client
	mailer       mailer.Mailer
}

// NewNotificationUsecase constructs a NotificationUsecase.
func NewNotificationUsecase(
	db *gorm.DB,
	notifRepo repository.NotificationRepository,
	tokenRepo repository.DeviceTokenRepository,
	processedRepo repository.ProcessedEventRepository,
	fbClient *firebase.Client,
	m mailer.Mailer,
) *NotificationUsecase {
	return &NotificationUsecase{
		db:            db,
		notifRepo:     notifRepo,
		tokenRepo:     tokenRepo,
		processedRepo: processedRepo,
		firebase:      fbClient,
		mailer:        m,
	}
}

// FanOutRequest carries everything needed to create and deliver one notification.
type FanOutRequest struct {
	EventID   string // Kafka event_id for idempotency
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      map[string]any
	Channel   entity.NotificationChannel
	EmailAddr string            // optional; email sent when non-empty and SMTP configured
	OrderID   string            // optional; triggers Firestore order-status write
	FSFields  map[string]any    // fields to merge into the Firestore order doc
}

// FanOut persists an in-app notification, optionally sends email, sends FCM push,
// and writes to Firestore. Idempotent via processed_events.
func (u *NotificationUsecase) FanOut(ctx context.Context, req FanOutRequest) error {
	if req.UserID == "" {
		return nil // cannot derive user — skip silently
	}

	dataJSON := json.RawMessage("{}")
	if len(req.Data) > 0 {
		b, err := json.Marshal(req.Data)
		if err != nil {
			slog.WarnContext(ctx, "notification: marshal data failed", "err", err)
		} else {
			dataJSON = b
		}
	}

	// Persist in-app notification inside a transaction that also marks the event processed.
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.EventID != "" {
			inserted, err := u.processedRepo.MarkProcessed(ctx, tx, req.EventID)
			if err != nil {
				return err
			}
			if !inserted {
				return nil // already processed — idempotent skip
			}
		}

		n := &entity.Notification{
			UserID:  req.UserID,
			Type:    req.Type,
			Title:   req.Title,
			Body:    req.Body,
			Data:    dataJSON,
			Channel: req.Channel,
		}
		return u.notifRepo.Create(ctx, n)
	})
	if err != nil {
		return err
	}

	// Best-effort side-effects outside the DB transaction.
	u.sendEmail(ctx, req)
	u.sendPush(ctx, req)
	u.writeFirestore(ctx, req)

	return nil
}

func (u *NotificationUsecase) sendEmail(ctx context.Context, req FanOutRequest) {
	if req.EmailAddr == "" || u.mailer == nil {
		return
	}
	if err := u.mailer.SendOTP(ctx, req.EmailAddr, req.Title, req.Body); err != nil {
		// SMTP may not be configured — log and continue.
		slog.WarnContext(ctx, "notification: email send failed", "to", req.EmailAddr, "err", err)
	}
}

func (u *NotificationUsecase) sendPush(ctx context.Context, req FanOutRequest) {
	if u.firebase == nil || !u.firebase.IsEnabled() {
		return
	}
	tokens, err := u.tokenRepo.ListByUser(ctx, req.UserID)
	if err != nil || len(tokens) == 0 {
		return
	}
	rawTokens := make([]string, len(tokens))
	for i, t := range tokens {
		rawTokens[i] = t.FCMToken
	}
	data := make(map[string]string)
	for k, v := range req.Data {
		if s, ok := v.(string); ok {
			data[k] = s
		}
	}
	if err := u.firebase.SendPush(ctx, rawTokens, req.Title, req.Body, data); err != nil {
		slog.WarnContext(ctx, "notification: FCM push failed", "user_id", req.UserID, "err", err)
	}
}

func (u *NotificationUsecase) writeFirestore(ctx context.Context, req FanOutRequest) {
	if u.firebase == nil || !u.firebase.IsEnabled() {
		return
	}
	if req.UserID == "" || req.OrderID == "" || len(req.FSFields) == 0 {
		return
	}
	if err := u.firebase.WriteOrderStatus(ctx, req.UserID, req.OrderID, req.FSFields); err != nil {
		slog.WarnContext(ctx, "notification: Firestore write failed",
			"user_id", req.UserID, "order_id", req.OrderID, "err", err)
	}
}

// ── Notification centre API ───────────────────────────────────────────────────

// ListResult is the paginated list response shape.
type ListResult struct {
	Items []*entity.Notification
	Total int64
}

// List returns paginated notifications for a user.
func (u *NotificationUsecase) List(ctx context.Context, userID string, skip, limit int) (*ListResult, error) {
	items, total, err := u.notifRepo.ListByUser(ctx, userID, skip, limit)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total}, nil
}

// UnreadCount returns the number of unread notifications for a user.
func (u *NotificationUsecase) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return u.notifRepo.UnreadCount(ctx, userID)
}

// MarkRead marks a single notification as read. Returns false if not found or not owned.
func (u *NotificationUsecase) MarkRead(ctx context.Context, id, userID string) (bool, error) {
	return u.notifRepo.MarkRead(ctx, id, userID)
}

// MarkAllRead marks all unread notifications for a user as read.
func (u *NotificationUsecase) MarkAllRead(ctx context.Context, userID string) error {
	return u.notifRepo.MarkAllRead(ctx, userID)
}

// RegisterToken upserts a device FCM token.
func (u *NotificationUsecase) RegisterToken(ctx context.Context, userID, fcmToken, platform string) error {
	return u.tokenRepo.Upsert(ctx, &entity.DeviceToken{
		UserID:   userID,
		FCMToken: fcmToken,
		Platform: platform,
	})
}

// DeleteToken removes a device token owned by userID. Returns false if not found.
func (u *NotificationUsecase) DeleteToken(ctx context.Context, id, userID string) (bool, error) {
	return u.tokenRepo.Delete(ctx, id, userID)
}

// ListTokensForUser returns all FCM tokens registered for a user.
// Used by tests to retrieve token IDs after registration.
func (u *NotificationUsecase) ListTokensForUser(ctx context.Context, userID string) ([]*entity.DeviceToken, error) {
	return u.tokenRepo.ListByUser(ctx, userID)
}
