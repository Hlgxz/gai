package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

const ContextKey = "session"

// Store persists session payloads.
type Store interface {
	Get(id string) (map[string]any, error)
	Set(id string, values map[string]any, ttl time.Duration) error
	Delete(id string) error
}

// MemoryStore is a process-local session store.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memItem
}

type memItem struct {
	values map[string]any
	expire time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]memItem)}
}

func (s *MemoryStore) Get(id string) (map[string]any, error) {
	s.mu.RLock()
	it, ok := s.items[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(it.expire) {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(it.values))
	for k, v := range it.values {
		out[k] = v
	}
	return out, nil
}

func (s *MemoryStore) Set(id string, values map[string]any, ttl time.Duration) error {
	s.mu.Lock()
	s.items[id] = memItem{values: values, expire: time.Now().Add(ttl)}
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.items, id)
	s.mu.Unlock()
	return nil
}

// Session is the per-request bag of values.
type Session struct {
	ID      string
	Values  map[string]any
	changed bool
}

func (s *Session) Get(key string) any { return s.Values[key] }

func (s *Session) Set(key string, value any) {
	s.Values[key] = value
	s.changed = true
}

func (s *Session) Forget(key string) {
	delete(s.Values, key)
	s.changed = true
}

func (s *Session) String(key string) string {
	v, _ := s.Values[key].(string)
	return v
}

// Manager issues session cookies and loads/saves via Store.
type Manager struct {
	Store      Store
	CookieName string
	TTL        time.Duration
	Secure     bool
}

func New(store Store) *Manager {
	return &Manager{
		Store:      store,
		CookieName: "gai_session",
		TTL:        24 * time.Hour,
	}
}

func (m *Manager) Middleware() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		id, _ := c.Cookie(m.CookieName)
		if id == "" {
			id = newSessionID()
			c.SetCookie(&http.Cookie{
				Name:     m.CookieName,
				Value:    id,
				Path:     "/",
				MaxAge:   int(m.TTL.Seconds()),
				HttpOnly: true,
				Secure:   m.Secure,
				SameSite: http.SameSiteLaxMode,
			})
		}
		values, _ := m.Store.Get(id)
		sess := &Session{ID: id, Values: values}
		c.Set(ContextKey, sess)
		c.Next()
		if sess.changed {
			_ = m.Store.Set(id, sess.Values, m.TTL)
		}
	}
}

func FromContext(c *ghttp.Context) *Session {
	v, ok := c.Get(ContextKey)
	if !ok {
		return nil
	}
	s, _ := v.(*Session)
	return s
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Encode is used by Redis-backed stores.
func Encode(values map[string]any) ([]byte, error) { return json.Marshal(values) }

func Decode(b []byte) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal(b, &m)
	if m == nil {
		m = map[string]any{}
	}
	return m, err
}
