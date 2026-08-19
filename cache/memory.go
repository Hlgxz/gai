package cache

import (
	"context"
	"sync"
	"time"
)

type memoryItem struct {
	value  []byte
	expire time.Time
}

// Memory is a process-local TTL cache.
type Memory struct {
	mu      sync.RWMutex
	items   map[string]memoryItem
	stop    chan struct{}
	stopped bool
}

// NewMemory creates an in-memory cache with background expiry cleanup.
func NewMemory() *Memory {
	m := &Memory{
		items: make(map[string]memoryItem),
		stop:  make(chan struct{}),
	}
	go m.janitor()
	return m
}

func (m *Memory) janitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			now := time.Now()
			m.mu.Lock()
			for k, it := range m.items {
				if !it.expire.IsZero() && now.After(it.expire) {
					delete(m.items, k)
				}
			}
			m.mu.Unlock()
		}
	}
}

// Stop terminates the cleanup goroutine.
func (m *Memory) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.stopped {
		close(m.stop)
		m.stopped = true
	}
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrMiss
	}
	if !it.expire.IsZero() && time.Now().After(it.expire) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return nil, ErrMiss
	}
	return it.value, nil
}

func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	it := memoryItem{value: append([]byte(nil), value...)}
	if ttl > 0 {
		it.expire = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = it
	m.mu.Unlock()
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

func (m *Memory) Has(ctx context.Context, key string) (bool, error) {
	_, err := m.Get(ctx, key)
	if err == ErrMiss {
		return false, nil
	}
	return err == nil, err
}
