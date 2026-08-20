package lock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotAcquired is returned when a lock is already held.
var ErrNotAcquired = errors.New("gai/lock: not acquired")

// Locker is a distributed (or process-local) mutex.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (release func(), err error)
}

// Memory is a process-local locker.
type Memory struct {
	mu   sync.Mutex
	held map[string]time.Time
}

func NewMemory() *Memory {
	return &Memory{held: make(map[string]time.Time)}
}

func (m *Memory) Acquire(_ context.Context, key string, ttl time.Duration) (func(), error) {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if exp, ok := m.held[key]; ok && now.Before(exp) {
		return nil, ErrNotAcquired
	}
	m.held[key] = now.Add(ttl)
	return func() {
		m.mu.Lock()
		delete(m.held, key)
		m.mu.Unlock()
	}, nil
}

// Redis is a SET NX EX locker.
type Redis struct {
	client redis.Cmdable
	prefix string
}

func NewRedis(client redis.Cmdable) *Redis {
	return &Redis{client: client, prefix: "gai:lock:"}
}

func (r *Redis) Acquire(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	ok, err := r.client.SetNX(ctx, r.prefix+key, "1", ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotAcquired
	}
	lockKey := r.prefix + key
	return func() {
		_ = r.client.Del(context.Background(), lockKey).Err()
	}, nil
}
