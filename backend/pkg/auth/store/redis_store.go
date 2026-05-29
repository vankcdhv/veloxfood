package store

import (
	"context"
	"fmt"
	"time"

	"project/pkg/config"

	goredis "github.com/redis/go-redis/v9"
)

const (
	jtiKeyPrefix  = "auth:jti:"
	userKeyPrefix = "auth:user:"
	userKeySuffix = ":jtis"
)

type redisAuthStore struct {
	client *goredis.Client
}

// NewRedisAuthStore creates its own Redis client from config — independent of pkg/cache.
// Uses a dedicated DB or the same DB; isolation via key prefix is sufficient for Phase 1.
func NewRedisAuthStore(cfg config.RedisConfig) (AuthStore, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("auth store redis ping: %w", err)
	}
	return &redisAuthStore{client: client}, nil
}

func jtiKey(jti string) string        { return jtiKeyPrefix + jti }
func userJTIsKey(uid string) string   { return userKeyPrefix + uid + userKeySuffix }

func (s *redisAuthStore) Whitelist(ctx context.Context, jti, userID string, ttl time.Duration) error {
	pipe := s.client.Pipeline()
	pipe.Set(ctx, jtiKey(jti), userID, ttl)
	pipe.SAdd(ctx, userJTIsKey(userID), jti)
	// user-set TTL = max refresh TTL; no strict expiry needed — RevokeAll clears it
	pipe.Expire(ctx, userJTIsKey(userID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *redisAuthStore) Exists(ctx context.Context, jti string) (bool, error) {
	n, err := s.client.Exists(ctx, jtiKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *redisAuthStore) Revoke(ctx context.Context, jti, userID string) error {
	pipe := s.client.Pipeline()
	pipe.Del(ctx, jtiKey(jti))
	pipe.SRem(ctx, userJTIsKey(userID), jti)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *redisAuthStore) RevokeAll(ctx context.Context, userID string) error {
	setKey := userJTIsKey(userID)
	jtis, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("smembers user jtis: %w", err)
	}
	if len(jtis) == 0 {
		return nil
	}
	keys := make([]string, len(jtis)+1)
	for i, jti := range jtis {
		keys[i] = jtiKey(jti)
	}
	keys[len(jtis)] = setKey
	return s.client.Del(ctx, keys...).Err()
}

// RotatePair atomically removes old pair, inserts new pair. Single pipeline = best-effort atomic.
// For strict atomicity a Lua script would be used; pipeline is sufficient for Phase 1.
func (s *redisAuthStore) RotatePair(ctx context.Context,
	oldJTIA, oldJTIR string,
	newJTIA, newJTIR, userID string,
	accessTTL, refreshTTL time.Duration,
) error {
	setKey := userJTIsKey(userID)
	pipe := s.client.Pipeline()
	// remove old
	pipe.Del(ctx, jtiKey(oldJTIA), jtiKey(oldJTIR))
	pipe.SRem(ctx, setKey, oldJTIA, oldJTIR)
	// insert new
	pipe.Set(ctx, jtiKey(newJTIA), userID, accessTTL)
	pipe.Set(ctx, jtiKey(newJTIR), userID, refreshTTL)
	pipe.SAdd(ctx, setKey, newJTIA, newJTIR)
	pipe.Expire(ctx, setKey, refreshTTL)
	_, err := pipe.Exec(ctx)
	return err
}
