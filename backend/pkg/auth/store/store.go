package store

import (
	"context"
	"time"
)

// AuthStore manages JWT JTI whitelisting per user.
// Whitelist model (not blacklist): a JTI must be present to be valid.
// Redis crash → all tokens invalid (fail-closed by design).
//
// Key layout:
//   auth:jti:<jti>         string = user_id, TTL mirrors token TTL
//   auth:user:<uid>:jtis   SET of active jtis for the user
type AuthStore interface {
	// Whitelist records jti as valid. Paired with SADD to user set.
	Whitelist(ctx context.Context, jti, userID string, ttl time.Duration) error

	// Exists returns true if jti is in the whitelist (has not been revoked/expired).
	Exists(ctx context.Context, jti string) (bool, error)

	// Revoke removes a single jti from both the key and user set.
	Revoke(ctx context.Context, jti, userID string) error

	// RevokeAll removes every active jti for userID — used on suspend/password change.
	RevokeAll(ctx context.Context, userID string) error

	// RotatePair atomically swaps old pair for new pair in Redis.
	RotatePair(ctx context.Context,
		oldJTIA, oldJTIR string,
		newJTIA, newJTIR, userID string,
		accessTTL, refreshTTL time.Duration,
	) error
}
