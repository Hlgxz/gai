package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Manager holds all configuration values loaded from YAML files and
// environment variables. It supports dot-notation access (e.g. "app.name").
type Manager struct {
	mu   sync.RWMutex
	data map[string]any
}

// New returns an empty configuration manager.
func New() *Manager {
	return &Manager{data: make(map[string]any)}
}

// Load reads YAML config files from the given directory. Each file becomes
// a top-level key (e.g. app.yaml -> config.Get("app.name")).
func (m *Manager) Load(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("gai/config: cannot read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		key := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		data, err := loadYAMLFile(dir + "/" + name)
		if err != nil {
			return fmt.Errorf("gai/config: failed to load %s: %w", name, err)
		}
		m.mu.Lock()
		m.data[key] = data
		m.mu.Unlock()
	}
	return nil
}

// Get retrieves a value by dot-notation key. Returns fallback if not found.
func (m *Manager) Get(key string, fallback ...any) any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parts := strings.Split(key, ".")
	var current any = m.data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return firstOr(fallback, nil)
			}
			current = val
		default:
			return firstOr(fallback, nil)
		}
	}

	if s, ok := current.(string); ok {
		return expandEnv(s)
	}
	return current
}

// GetString is a typed convenience wrapper around Get.
func (m *Manager) GetString(key string, fallback ...string) string {
	fb := make([]any, len(fallback))
	for i, v := range fallback {
		fb[i] = v
	}
	val := m.Get(key, fb...)
	if val == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetInt is a typed convenience wrapper around Get.
func (m *Manager) GetInt(key string, fallback ...int) int {
	val := m.Get(key)
	if val == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
}

// GetBool is a typed convenience wrapper around Get.
func (m *Manager) GetBool(key string, fallback ...bool) bool {
	val := m.Get(key)
	if val == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return false
}

// Set puts a value at the given dot-notation key.
func (m *Manager) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	parts := strings.Split(key, ".")
	current := m.data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part]
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		if nested, ok := next.(map[string]any); ok {
			current = nested
		} else {
			nm := make(map[string]any)
			current[part] = nm
			current = nm
		}
	}
}

// All returns a shallow copy of the top-level configuration map.
func (m *Manager) All() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]any, len(m.data))
	for k, v := range m.data {
		out[k] = v
	}
	return out
}

// expandEnv replaces ${VAR} or ${VAR:default} patterns with environment values.
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":"); idx != -1 {
			envKey := key[:idx]
			def := key[idx+1:]
			if val, ok := os.LookupEnv(envKey); ok {
				return val
			}
			return def
		}
		return os.Getenv(key)
	})
}

func firstOr[T any](slice []T, fallback T) T {
	if len(slice) > 0 {
		return slice[0]
	}
	return fallback
}
