package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis wraps a go-redis client.
type Redis struct {
	client redis.Cmdable
}

// NewRedis creates a Redis-backed cache from a go-redis client.
func NewRedis(client redis.Cmdable) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrMiss
	}
	return b, err
}

func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *Redis) Has(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}
