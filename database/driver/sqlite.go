package driver

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func init() {
	Register(&SQLite{})
}

// SQLite implements the Driver interface for SQLite (via modernc.org/sqlite,
// a pure-Go driver with no cgo requirement).
type SQLite struct{}

func (SQLite) Name() string { return "sqlite" }

func (SQLite) Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}

func (SQLite) Placeholder(_ int) string { return "?" }

func (SQLite) QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (SQLite) AutoIncrementType() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (SQLite) ColumnType(gaiType string, size int) string {
	switch gaiType {
	case "string":
		if size <= 0 {
			size = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", size)
	case "text":
		return "TEXT"
	case "int", "integer", "bigint":
		return "INTEGER"
	case "float", "double":
		return "REAL"
	case "decimal":
		return "REAL"
	case "bool", "boolean":
		return "INTEGER"
	case "date", "datetime", "timestamp":
		return "TEXT"
	case "json":
		return "TEXT"
	case "enum":
		return "VARCHAR(50)"
	default:
		return "TEXT"
	}
}
