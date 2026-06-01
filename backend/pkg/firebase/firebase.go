// Package firebase wraps Firebase Admin SDK for the notification service.
// When ServiceAccountPath is empty or the file cannot be loaded, a no-op client
// is returned — the service boots normally and logs a warning on each skipped call.
package firebase

import (
	"context"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"

	"project/pkg/config"
)

// Client wraps Firebase Firestore + FCM. All methods are no-ops when disabled.
type Client struct {
	enabled   bool
	firestore *firestore.Client
	messaging *messaging.Client
	projectID string
}

// New initialises a Firebase app from the service-account JSON at cfg.ServiceAccountPath.
// If the path is empty or init fails, a disabled (no-op) client is returned — never an error.
func New(cfg config.FirebaseConfig) (*Client, error) {
	if cfg.ServiceAccountPath == "" {
		slog.Warn("firebase: service_account_path is empty — Firestore/FCM disabled")
		return &Client{enabled: false}, nil
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID},
		option.WithCredentialsFile(cfg.ServiceAccountPath))
	if err != nil {
		slog.Warn("firebase: failed to initialise app — Firestore/FCM disabled", "err", err)
		return &Client{enabled: false}, nil
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		slog.Warn("firebase: failed to create Firestore client — Firestore/FCM disabled", "err", err)
		return &Client{enabled: false}, nil
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		// Firestore may still work; disable both for simplicity.
		_ = fsClient.Close()
		slog.Warn("firebase: failed to create Messaging client — Firestore/FCM disabled", "err", err)
		return &Client{enabled: false}, nil
	}

	slog.Info("firebase: client initialised", "project_id", cfg.ProjectID)
	return &Client{
		enabled:   true,
		firestore: fsClient,
		messaging: msgClient,
		projectID: cfg.ProjectID,
	}, nil
}

// IsEnabled reports whether Firebase is available.
func (c *Client) IsEnabled() bool { return c.enabled }

// WriteOrderStatus merges fields into Firestore document orders/{userID}/orders/{orderID}.
// The FE subscribes to this path for realtime order tracking.
// No-op when disabled.
func (c *Client) WriteOrderStatus(ctx context.Context, userID, orderID string, fields map[string]any) error {
	if !c.enabled {
		slog.DebugContext(ctx, "firebase: WriteOrderStatus skipped (disabled)",
			"user_id", userID, "order_id", orderID)
		return nil
	}

	docRef := c.firestore.
		Collection("orders").Doc(userID).
		Collection("orders").Doc(orderID)

	if _, err := docRef.Set(ctx, fields, firestore.MergeAll); err != nil {
		return fmt.Errorf("firebase: WriteOrderStatus %s/%s: %w", userID, orderID, err)
	}
	return nil
}

// SendPush sends an FCM multicast push to the given tokens.
// Partial failures are logged but not returned as an error — best-effort delivery.
// No-op when disabled or tokens list is empty.
func (c *Client) SendPush(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if !c.enabled || len(tokens) == 0 {
		return nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := c.messaging.SendEachForMulticast(ctx, msg)
	if err != nil {
		return fmt.Errorf("firebase: SendPush multicast: %w", err)
	}

	if resp.FailureCount > 0 {
		for i, r := range resp.Responses {
			if !r.Success {
				slog.WarnContext(ctx, "firebase: push delivery failed",
					"token_index", i, "err", r.Error)
			}
		}
	}
	return nil
}

// Close releases underlying Firestore connection. Safe to call when disabled.
func (c *Client) Close() {
	if c.enabled && c.firestore != nil {
		_ = c.firestore.Close()
	}
}
