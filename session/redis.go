package session

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore persists sessions in Redis.
type RedisStore struct {
	client redis.Cmdable
	prefix string
}

func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client, prefix: "gai:session:"}
}

func (s *RedisStore) Get(id string) (map[string]any, error) {
	b, err := s.client.Get(context.Background(), s.prefix+id).Bytes()
	if err == redis.Nil {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

func (s *RedisStore) Set(id string, values map[string]any, ttl time.Duration) error {
	b, err := Encode(values)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), s.prefix+id, b, ttl).Err()
}

func (s *RedisStore) Delete(id string) error {
	return s.client.Del(context.Background(), s.prefix+id).Err()
}
