package migration

import (
	"fmt"
	"strings"

	"github.com/Hlgxz/gai/database/driver"
)

// Blueprint describes a table schema, similar to Laravel's Schema\Blueprint.
type Blueprint struct {
	table   string
	columns []Column
	driver  driver.Driver
}

// Column represents a single column definition in a migration.
type Column struct {
	Name          string
	Type          string
	Size          int
	Nullable      bool
	Unique        bool
	Index         bool
	Default       string
	PrimaryKey    bool
	AutoIncrement bool
}

// NewBlueprint creates a new Blueprint for the given table.
func NewBlueprint(table string, drv driver.Driver) *Blueprint {
	return &Blueprint{table: table, driver: drv}
}

// ID adds an auto-incrementing big integer primary key.
func (b *Blueprint) ID() *Blueprint {
	b.columns = append(b.columns, Column{
		Name:          "id",
		PrimaryKey:    true,
		AutoIncrement: true,
	})
	return b
}

// String adds a VARCHAR column.
func (b *Blueprint) String(name string, size int) *Column {
	col := Column{Name: name, Type: "string", Size: size}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Text adds a TEXT column.
func (b *Blueprint) Text(name string) *Column {
	col := Column{Name: name, Type: "text"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Integer adds an INT column.
func (b *Blueprint) Integer(name string) *Column {
	col := Column{Name: name, Type: "int"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// BigInteger adds a BIGINT column.
func (b *Blueprint) BigInteger(name string) *Column {
	col := Column{Name: name, Type: "bigint"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Float adds a FLOAT column.
func (b *Blueprint) Float(name string) *Column {
	col := Column{Name: name, Type: "float"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Decimal adds a DECIMAL column.
func (b *Blueprint) Decimal(name string) *Column {
	col := Column{Name: name, Type: "decimal"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Boolean adds a BOOLEAN column.
func (b *Blueprint) Boolean(name string) *Column {
	col := Column{Name: name, Type: "bool"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// DateTime adds a DATETIME/TIMESTAMP column.
func (b *Blueprint) DateTime(name string) *Column {
	col := Column{Name: name, Type: "datetime"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// JSON adds a JSON column.
func (b *Blueprint) JSON(name string) *Column {
	col := Column{Name: name, Type: "json"}
	b.columns = append(b.columns, col)
	return &b.columns[len(b.columns)-1]
}

// Timestamps adds created_at and updated_at columns.
func (b *Blueprint) Timestamps() *Blueprint {
	b.DateTime("created_at")
	b.DateTime("updated_at").SetNullable()
	return b
}

// SoftDeletes adds a nullable deleted_at column.
func (b *Blueprint) SoftDeletes() *Blueprint {
	b.DateTime("deleted_at").SetNullable()
	return b
}

// Column chainable modifiers.

func (c *Column) SetNullable() *Column {
	c.Nullable = true
	return c
}

func (c *Column) SetUnique() *Column {
	c.Unique = true
	return c
}

func (c *Column) SetIndex() *Column {
	c.Index = true
	return c
}

func (c *Column) SetDefault(val string) *Column {
	c.Default = val
	return c
}

// ---------------------------------------------------------- SQL generation

// ToCreateSQL generates the CREATE TABLE statement.
func (b *Blueprint) ToCreateSQL() string {
	var defs []string
	var indexes []string

	for _, col := range b.columns {
		if col.AutoIncrement {
			defs = append(defs, fmt.Sprintf("  %s %s",
				b.driver.QuoteIdent(col.Name),
				b.driver.AutoIncrementType()))
			continue
		}

		def := fmt.Sprintf("  %s %s",
			b.driver.QuoteIdent(col.Name),
			b.driver.ColumnType(col.Type, col.Size))

		if !col.Nullable {
			def += " NOT NULL"
		}
		if col.Default != "" {
			def += " DEFAULT " + col.Default
		}
		if col.Unique {
			def += " UNIQUE"
		}
		defs = append(defs, def)

		if col.Index {
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s);",
				b.table, col.Name, b.driver.QuoteIdent(b.table), b.driver.QuoteIdent(col.Name)))
		}
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		b.driver.QuoteIdent(b.table),
		strings.Join(defs, ",\n"))

	if len(indexes) > 0 {
		sql += "\n" + strings.Join(indexes, "\n")
	}

	return sql
}

// ToDropSQL generates the DROP TABLE statement.
func (b *Blueprint) ToDropSQL() string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", b.driver.QuoteIdent(b.table))
}
