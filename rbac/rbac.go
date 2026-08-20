package rbac

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/redis/go-redis/v9"
)

// Store persists role assignments and permissions.
type Store interface {
	AssignRole(userID, role string) error
	RevokeRole(userID, role string) error
	Grant(role, permission string) error
	Deny(role, permission string) error
	HasRole(userID, role string) bool
	Can(userID, permission string) bool
}

// MemoryStore is the default in-process store.
type MemoryStore struct {
	mu        sync.RWMutex
	userRoles map[string]map[string]struct{}
	rolePerms map[string]map[string]struct{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		userRoles: make(map[string]map[string]struct{}),
		rolePerms: make(map[string]map[string]struct{}),
	}
}

func (s *MemoryStore) AssignRole(userID, role string) error {
	s.mu.Lock()
	if s.userRoles[userID] == nil {
		s.userRoles[userID] = make(map[string]struct{})
	}
	s.userRoles[userID][role] = struct{}{}
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) RevokeRole(userID, role string) error {
	s.mu.Lock()
	delete(s.userRoles[userID], role)
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Grant(role, permission string) error {
	s.mu.Lock()
	if s.rolePerms[role] == nil {
		s.rolePerms[role] = make(map[string]struct{})
	}
	s.rolePerms[role][permission] = struct{}{}
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Deny(role, permission string) error {
	s.mu.Lock()
	delete(s.rolePerms[role], permission)
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) HasRole(userID, role string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.userRoles[userID][role]
	return ok
}

func (s *MemoryStore) Can(userID, permission string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for role := range s.userRoles[userID] {
		if _, ok := s.rolePerms[role][permission]; ok {
			return true
		}
		if _, ok := s.rolePerms[role]["*"]; ok {
			return true
		}
	}
	return false
}

// RedisStore persists RBAC in Redis sets.
type RedisStore struct {
	client redis.Cmdable
	prefix string
}

func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client, prefix: "gai:rbac:"}
}

func (s *RedisStore) userKey(id string) string   { return s.prefix + "user:" + id }
func (s *RedisStore) roleKey(role string) string { return s.prefix + "role:" + role }

func ctxBG() context.Context { return context.Background() }

func (s *RedisStore) AssignRole(userID, role string) error {
	return s.client.SAdd(ctxBG(), s.userKey(userID), role).Err()
}

func (s *RedisStore) RevokeRole(userID, role string) error {
	return s.client.SRem(ctxBG(), s.userKey(userID), role).Err()
}

func (s *RedisStore) Grant(role, permission string) error {
	return s.client.SAdd(ctxBG(), s.roleKey(role), permission).Err()
}

func (s *RedisStore) Deny(role, permission string) error {
	return s.client.SRem(ctxBG(), s.roleKey(role), permission).Err()
}

func (s *RedisStore) HasRole(userID, role string) bool {
	ok, _ := s.client.SIsMember(ctxBG(), s.userKey(userID), role).Result()
	return ok
}

func (s *RedisStore) Can(userID, permission string) bool {
	roles, err := s.client.SMembers(ctxBG(), s.userKey(userID)).Result()
	if err != nil {
		return false
	}
	for _, role := range roles {
		ok, _ := s.client.SIsMember(ctxBG(), s.roleKey(role), permission).Result()
		if ok {
			return true
		}
		star, _ := s.client.SIsMember(ctxBG(), s.roleKey(role), "*").Result()
		if star {
			return true
		}
	}
	return false
}

// Manager is an RBAC enforcer backed by a Store.
type Manager struct {
	store Store
}

func New() *Manager {
	return NewWithStore(NewMemoryStore())
}

func NewWithStore(store Store) *Manager {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Manager{store: store}
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
	_ = m.store.AssignRole(uid(userID), role)
}

func (m *Manager) RevokeRole(userID any, role string) {
	_ = m.store.RevokeRole(uid(userID), role)
}

func (m *Manager) Grant(role, permission string) {
	_ = m.store.Grant(role, permission)
}

func (m *Manager) Deny(role, permission string) {
	_ = m.store.Deny(role, permission)
}

func (m *Manager) HasRole(userID any, role string) bool {
	return m.store.HasRole(uid(userID), role)
}

func (m *Manager) Can(userID any, permission string) bool {
	return m.store.Can(uid(userID), permission)
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
