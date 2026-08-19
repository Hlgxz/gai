package database

import (
	"fmt"
	"time"

	"github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/orm"
)

// Config describes how to open a database connection.
type Config struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open creates an ORM database handle from a driver name and DSN.
func Open(cfg Config) (*orm.DB, error) {
	if cfg.Driver == "" {
		cfg.Driver = "sqlite"
	}
	if cfg.DSN == "" {
		cfg.DSN = ":memory:"
	}

	drv, err := driver.Get(cfg.Driver)
	if err != nil {
		return nil, err
	}

	sqlDB, err := drv.Open(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("gai/database: open %s: %w", cfg.Driver, err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("gai/database: ping %s: %w", cfg.Driver, err)
	}

	db := &orm.DB{
		SQL:        sqlDB,
		DriverName: drv.Name(),
		QuoteIdent: drv.QuoteIdent,
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	db.ConfigurePool(maxOpen, maxIdle, lifetime)

	return db, nil
}

// Manager holds named database connections (read/write splitting, extra DBs).
type Manager struct {
	conns       map[string]*orm.DB
	defaultName string
}

// NewManager creates an empty connection manager.
func NewManager() *Manager {
	return &Manager{conns: make(map[string]*orm.DB)}
}

// Add registers a named connection. The first added connection becomes default.
func (m *Manager) Add(name string, db *orm.DB) {
	m.conns[name] = db
	if m.defaultName == "" {
		m.defaultName = name
	}
}

// SetDefault selects the default connection name.
func (m *Manager) SetDefault(name string) {
	m.defaultName = name
}

// Connection returns a named connection, or the default if name is empty.
func (m *Manager) Connection(name ...string) (*orm.DB, error) {
	key := m.defaultName
	if len(name) > 0 && name[0] != "" {
		key = name[0]
	}
	db, ok := m.conns[key]
	if !ok {
		return nil, fmt.Errorf("gai/database: connection %q not registered", key)
	}
	return db, nil
}

// Default returns the default connection, or nil if none is registered.
func (m *Manager) Default() *orm.DB {
	db, _ := m.Connection()
	return db
}

// Close closes all registered connections.
func (m *Manager) Close() error {
	var first error
	for name, db := range m.conns {
		if err := db.Close(); err != nil && first == nil {
			first = fmt.Errorf("gai/database: close %s: %w", name, err)
		}
	}
	return first
}
