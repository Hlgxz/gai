package auth

import (
	"fmt"
	"net/http"
	"sync"

	ghttp "github.com/Hlgxz/gai/http"
)

// Manager manages multiple authentication guards, similar to Laravel's
// Auth facade with support for multiple guard drivers.
type Manager struct {
	mu       sync.RWMutex
	guards   map[string]Guard
	fallback string
}

// NewManager creates an auth manager with the given default guard name.
func NewManager(defaultGuard string) *Manager {
	return &Manager{
		guards:   make(map[string]Guard),
		fallback: defaultGuard,
	}
}

// RegisterGuard adds a guard to the manager.
func (m *Manager) RegisterGuard(guard Guard) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guards[guard.Name()] = guard
}

// Guard returns a named guard, or the default if name is empty.
func (m *Manager) Guard(name ...string) (Guard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.fallback
	if len(name) > 0 && name[0] != "" {
		key = name[0]
	}

	g, ok := m.guards[key]
	if !ok {
		return nil, fmt.Errorf("gai/auth: guard %q not registered", key)
	}
	return g, nil
}

// Middleware returns an HTTP handler that enforces authentication using the
// specified guard(s). If no guard is specified, the default is used.
func (m *Manager) Middleware(guardNames ...string) ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		names := guardNames
		if len(names) == 0 {
			names = []string{m.fallback}
		}

		for _, name := range names {
			g, err := m.Guard(name)
			if err != nil {
				c.AbortWithJSON(http.StatusUnauthorized, map[string]any{
					"code":    401,
					"message": "Unauthorized",
				})
				return
			}
			if g.Check(c) {
				c.Set("auth_guard", name)
				c.Next()
				return
			}
		}

		c.AbortWithJSON(http.StatusUnauthorized, map[string]any{
			"code":    401,
			"message": "Unauthorized",
		})
	}
}
