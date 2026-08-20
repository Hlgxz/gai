package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Hlgxz/gai/lock"
)

// ErrMiss is returned when a key is not present.
var ErrMiss = errors.New("gai/cache: miss")

// Store is a cache backend.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Has(ctx context.Context, key string) (bool, error)
}

// Manager wraps a Store with helpers.
type Manager struct {
	Store  Store
	Prefix string
	Locker lock.Locker

	onceMu sync.Mutex
}

func (m *Manager) key(k string) string {
	if m.Prefix == "" {
		return k
	}
	return m.Prefix + k
}

func (m *Manager) Get(ctx context.Context, key string) ([]byte, error) {
	return m.Store.Get(ctx, m.key(key))
}

func (m *Manager) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return m.Store.Set(ctx, m.key(key), value, ttl)
}

func (m *Manager) Delete(ctx context.Context, key string) error {
	return m.Store.Delete(ctx, m.key(key))
}

func (m *Manager) Has(ctx context.Context, key string) (bool, error) {
	return m.Store.Has(ctx, m.key(key))
}

func (m *Manager) GetString(ctx context.Context, key string) (string, error) {
	b, err := m.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *Manager) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.Set(ctx, key, []byte(value), ttl)
}

// RememberJSON loads key or computes fn, stores the JSON result, and returns it.
func RememberJSON[T any](m *Manager, ctx context.Context, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	b, err := m.Get(ctx, key)
	if err == nil {
		if err := json.Unmarshal(b, &zero); err == nil {
			return zero, nil
		}
	} else if !errors.Is(err, ErrMiss) {
		return zero, err
	}
	val, err := fn()
	if err != nil {
		return zero, err
	}
	raw, err := json.Marshal(val)
	if err != nil {
		return zero, err
	}
	if err := m.Set(ctx, key, raw, ttl); err != nil {
		return val, err
	}
	return val, nil
}

func (m *Manager) locker() lock.Locker {
	if m.Locker != nil {
		return m.Locker
	}
	m.onceMu.Lock()
	defer m.onceMu.Unlock()
	if m.Locker == nil {
		m.Locker = lock.NewMemory()
	}
	return m.Locker
}

// Lock acquires a named lock (process-local by default, or Redis when Locker is set).
func (m *Manager) Lock(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	return m.locker().Acquire(ctx, m.key(key), ttl)
}

// Once runs fn only if key is absent, then stores a marker for ttl.
func (m *Manager) Once(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	ok, err := m.Has(ctx, key)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	rel, err := m.Lock(ctx, "once:"+key, ttl)
	if err != nil {
		if errors.Is(err, lock.ErrNotAcquired) {
			return nil
		}
		return err
	}
	defer rel()
	ok, err = m.Has(ctx, key)
	if err != nil || ok {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	return m.Set(ctx, key, []byte("1"), ttl)
}
