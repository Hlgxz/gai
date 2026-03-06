package driver

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func init() {
	Register(&Postgres{})
}

// Postgres implements the Driver interface for PostgreSQL.
type Postgres struct{}

func (Postgres) Name() string { return "postgres" }

func (Postgres) Open(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}

func (Postgres) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (Postgres) QuoteIdent(name string) string {
	return `"` + name + `"`
}

func (Postgres) AutoIncrementType() string {
	return "BIGSERIAL PRIMARY KEY"
}

func (Postgres) ColumnType(gaiType string, size int) string {
	switch gaiType {
	case "string":
		if size <= 0 {
			size = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", size)
	case "text":
		return "TEXT"
	case "int", "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "float":
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
	case "decimal":
		return "NUMERIC(10,2)"
	case "bool", "boolean":
		return "BOOLEAN"
	case "date":
		return "DATE"
	case "datetime", "timestamp":
		return "TIMESTAMP"
	case "json":
		return "JSONB"
	case "enum":
		return "VARCHAR(50)"
	default:
		return "VARCHAR(255)"
	}
}
