package driver

import (
	"database/sql"
	"fmt"
)

// Driver abstracts the differences between database engines so the ORM
// and migration layers can work across MySQL, PostgreSQL, and SQLite.
type Driver interface {
	// Name returns the driver identifier (e.g. "mysql", "postgres", "sqlite").
	Name() string

	// Open creates a *sql.DB connection from a DSN string.
	Open(dsn string) (*sql.DB, error)

	// Placeholder returns the parameter placeholder for the n-th argument.
	// MySQL uses ?, PostgreSQL uses $1/$2, SQLite uses ?.
	Placeholder(n int) string

	// QuoteIdent quotes a column or table identifier.
	QuoteIdent(name string) string

	// AutoIncrementType returns the SQL type for an auto-increment primary key.
	AutoIncrementType() string

	// ColumnType maps a generic Gai type name to the SQL column type.
	ColumnType(gaiType string, size int) string
}

// Registry holds all registered database drivers.
var registry = map[string]Driver{}

// Register makes a driver available by name.
func Register(d Driver) {
	registry[d.Name()] = d
}

// Get returns a registered driver by name.
func Get(name string) (Driver, error) {
	d, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("gai/database: unknown driver %q", name)
	}
	return d, nil
}
