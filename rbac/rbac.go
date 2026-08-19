package rbac

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	ghttp "github.com/Hlgxz/gai/http"
)

// Manager is an in-memory RBAC enforcer. Persist assignments yourself
// (or wrap this with a database-backed store).
type Manager struct {
	mu        sync.RWMutex
	userRoles map[string]map[string]struct{}
	rolePerms map[string]map[string]struct{}
}

func New() *Manager {
	return &Manager{
		userRoles: make(map[string]map[string]struct{}),
		rolePerms: make(map[string]map[string]struct{}),
	}
}

func uid(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case uint64:
		return strconv.FormatUint(v, 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(id)
	}
}

func (m *Manager) AssignRole(userID any, role string) {
	key := uid(userID)
	m.mu.Lock()
	if m.userRoles[key] == nil {
		m.userRoles[key] = make(map[string]struct{})
	}
	m.userRoles[key][role] = struct{}{}
	m.mu.Unlock()
}

func (m *Manager) RevokeRole(userID any, role string) {
	m.mu.Lock()
	delete(m.userRoles[uid(userID)], role)
	m.mu.Unlock()
}

func (m *Manager) Grant(role, permission string) {
	m.mu.Lock()
	if m.rolePerms[role] == nil {
		m.rolePerms[role] = make(map[string]struct{})
	}
	m.rolePerms[role][permission] = struct{}{}
	m.mu.Unlock()
}

func (m *Manager) Deny(role, permission string) {
	m.mu.Lock()
	delete(m.rolePerms[role], permission)
	m.mu.Unlock()
}

func (m *Manager) HasRole(userID any, role string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.userRoles[uid(userID)][role]
	return ok
}

func (m *Manager) Can(userID any, permission string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for role := range m.userRoles[uid(userID)] {
		if _, ok := m.rolePerms[role][permission]; ok {
			return true
		}
		if _, ok := m.rolePerms[role]["*"]; ok {
			return true
		}
	}
	return false
}

// Middleware requires the authenticated user (auth_user_id) to have permission.
func (m *Manager) Middleware(permission string) ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		raw, ok := c.Get("auth_user_id")
		if !ok || !m.Can(raw, permission) {
			c.AbortWithJSON(http.StatusForbidden, map[string]any{
				"code":    403,
				"message": "Forbidden",
			})
			return
		}
		c.Next()
	}
}
