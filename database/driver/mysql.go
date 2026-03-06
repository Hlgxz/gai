package driver

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	Register(&MySQL{})
}

// MySQL implements the Driver interface for MySQL / MariaDB.
type MySQL struct{}

func (MySQL) Name() string { return "mysql" }

func (MySQL) Open(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

func (MySQL) Placeholder(_ int) string { return "?" }

func (MySQL) QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func (MySQL) AutoIncrementType() string {
	return "BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY"
}

func (MySQL) ColumnType(gaiType string, size int) string {
	switch gaiType {
	case "string":
		if size <= 0 {
			size = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", size)
	case "text":
		return "TEXT"
	case "int", "integer":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "float":
		return "FLOAT"
	case "double":
		return "DOUBLE"
	case "decimal":
		return "DECIMAL(10,2)"
	case "bool", "boolean":
		return "TINYINT(1)"
	case "date":
		return "DATE"
	case "datetime", "timestamp":
		return "DATETIME"
	case "json":
		return "JSON"
	case "enum":
		return "VARCHAR(50)"
	default:
		return "VARCHAR(255)"
	}
}
