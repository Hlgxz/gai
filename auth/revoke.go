package auth

import (
	"sync"
	"time"
)

// Revoker tracks revoked JWT IDs (jti).
type Revoker interface {
	Revoke(jti string, until time.Time)
	Revoked(jti string) bool
}

// MemoryRevoker is a process-local token blacklist.
type MemoryRevoker struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

// NewMemoryRevoker creates an empty in-memory blacklist.
func NewMemoryRevoker() *MemoryRevoker {
	return &MemoryRevoker{items: make(map[string]time.Time)}
}

func (r *MemoryRevoker) Revoke(jti string, until time.Time) {
	r.mu.Lock()
	r.items[jti] = until
	r.mu.Unlock()
}

func (r *MemoryRevoker) Revoked(jti string) bool {
	r.mu.RLock()
	until, ok := r.items[jti]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(until) {
		r.mu.Lock()
		delete(r.items, jti)
		r.mu.Unlock()
		return false
	}
	return true
}
