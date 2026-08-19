package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRevoker stores revoked JWT IDs in Redis.
type RedisRevoker struct {
	client redis.Cmdable
	prefix string
}

func NewRedisRevoker(client redis.Cmdable) *RedisRevoker {
	return &RedisRevoker{client: client, prefix: "gai:jwt:revoked:"}
}

func (r *RedisRevoker) Revoke(jti string, until time.Time) {
	ttl := time.Until(until)
	if ttl <= 0 {
		return
	}
	_ = r.client.Set(context.Background(), r.prefix+jti, "1", ttl).Err()
}

func (r *RedisRevoker) Revoked(jti string) bool {
	n, err := r.client.Exists(context.Background(), r.prefix+jti).Result()
	return err == nil && n > 0
}
