package migration

import (
	"fmt"
	"strings"

	"github.com/Hlgxz/gai/database/driver"
)

// Blueprint describes a table schema, similar to Laravel's Schema\Blueprint.
type Blueprint struct {
	table        string
	columns      []Column
	indexes      []IndexDef
	foreignKeys  []*ForeignKey
	dropColumns  []string
	dropIndexes  []string
	dropForeigns []string
	driver       driver.Driver
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

// IndexDef is a (possibly composite) index.
type IndexDef struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKey describes a foreign key constraint.
type ForeignKey struct {
	Name      string
	Column    string
	RefTable  string
	RefColumn string
	Delete    string
	Update    string
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

// Index adds a (possibly composite) index on the given columns.
func (b *Blueprint) Index(columns ...string) *Blueprint {
	if len(columns) == 0 {
		return b
	}
	b.indexes = append(b.indexes, IndexDef{
		Name:    b.indexName("idx", columns),
		Columns: columns,
	})
	return b
}

// UniqueIndex adds a unique (possibly composite) index.
func (b *Blueprint) UniqueIndex(columns ...string) *Blueprint {
	if len(columns) == 0 {
		return b
	}
	b.indexes = append(b.indexes, IndexDef{
		Name:    b.indexName("uniq", columns),
		Columns: columns,
		Unique:  true,
	})
	return b
}

// Foreign starts a foreign key on column. Chain References/On/OnDelete/OnUpdate.
func (b *Blueprint) Foreign(column string) *ForeignKey {
	fk := &ForeignKey{
		Name:      fmt.Sprintf("fk_%s_%s", b.table, column),
		Column:    column,
		RefColumn: "id",
	}
	b.foreignKeys = append(b.foreignKeys, fk)
	return fk
}

// References sets the referenced column (default "id").
func (fk *ForeignKey) References(column string) *ForeignKey {
	fk.RefColumn = column
	return fk
}

// On sets the referenced table.
func (fk *ForeignKey) On(table string) *ForeignKey {
	fk.RefTable = table
	return fk
}

// OnDelete sets the ON DELETE action (CASCADE, SET NULL, RESTRICT, ...).
func (fk *ForeignKey) OnDelete(action string) *ForeignKey {
	fk.Delete = action
	return fk
}

// OnUpdate sets the ON UPDATE action.
func (fk *ForeignKey) OnUpdate(action string) *ForeignKey {
	fk.Update = action
	return fk
}

// DropColumn queues a column drop for ToAlterSQL.
func (b *Blueprint) DropColumn(name string) *Blueprint {
	b.dropColumns = append(b.dropColumns, name)
	return b
}

// DropIndex queues an index drop for ToAlterSQL.
func (b *Blueprint) DropIndex(name string) *Blueprint {
	b.dropIndexes = append(b.dropIndexes, name)
	return b
}

// DropForeign queues a foreign key drop for ToAlterSQL.
func (b *Blueprint) DropForeign(name string) *Blueprint {
	b.dropForeigns = append(b.dropForeigns, name)
	return b
}

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

func (b *Blueprint) q(name string) string {
	return b.driver.QuoteIdent(name)
}

func (b *Blueprint) indexName(prefix string, columns []string) string {
	return prefix + "_" + b.table + "_" + strings.Join(columns, "_")
}

func (b *Blueprint) columnDefinition(col Column) string {
	if col.AutoIncrement {
		return fmt.Sprintf("%s %s", b.q(col.Name), b.driver.AutoIncrementType())
	}
	def := fmt.Sprintf("%s %s", b.q(col.Name), b.driver.ColumnType(col.Type, col.Size))
	if !col.Nullable {
		def += " NOT NULL"
	}
	if col.Default != "" {
		def += " DEFAULT " + col.Default
	}
	if col.Unique {
		def += " UNIQUE"
	}
	return def
}

func (b *Blueprint) createIndexSQL(idx IndexDef) string {
	kind := "INDEX"
	if idx.Unique {
		kind = "UNIQUE INDEX"
	}
	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = b.q(c)
	}
	ifExists := ""
	if b.driver.Name() != "mysql" {
		ifExists = "IF NOT EXISTS "
	}
	return fmt.Sprintf("CREATE %s %s%s ON %s (%s);",
		kind, ifExists, b.q(idx.Name), b.q(b.table), strings.Join(cols, ", "))
}

func (b *Blueprint) fkConstraintSQL(fk *ForeignKey) string {
	sql := fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		b.q(fk.Name), b.q(fk.Column), b.q(fk.RefTable), b.q(fk.RefColumn))
	if fk.Delete != "" {
		sql += " ON DELETE " + fk.Delete
	}
	if fk.Update != "" {
		sql += " ON UPDATE " + fk.Update
	}
	return sql
}

// ToCreateSQL generates the CREATE TABLE statement plus indexes.
func (b *Blueprint) ToCreateSQL() string {
	var defs []string
	var indexes []string

	for _, col := range b.columns {
		defs = append(defs, "  "+b.columnDefinition(col))
		if col.Index {
			indexes = append(indexes, b.createIndexSQL(IndexDef{
				Name:    b.indexName("idx", []string{col.Name}),
				Columns: []string{col.Name},
			}))
		}
	}

	for _, fk := range b.foreignKeys {
		if fk.RefTable == "" {
			continue
		}
		defs = append(defs, "  "+b.fkConstraintSQL(fk))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		b.q(b.table),
		strings.Join(defs, ",\n"))

	for _, idx := range b.indexes {
		indexes = append(indexes, b.createIndexSQL(idx))
	}

	if len(indexes) > 0 {
		sql += "\n" + strings.Join(indexes, "\n")
	}

	return sql
}

// ToDropSQL generates the DROP TABLE statement.
func (b *Blueprint) ToDropSQL() string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", b.q(b.table))
}

// ToAlterSQL generates ALTER TABLE / CREATE INDEX / DROP statements.
// Columns added via String/Integer/... become ADD COLUMN; use DropColumn for drops.
func (b *Blueprint) ToAlterSQL() string {
	var stmts []string
	drv := b.driver.Name()

	for _, col := range b.columns {
		if col.AutoIncrement {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", b.q(b.table), b.columnDefinition(col)))
		if col.Index {
			stmts = append(stmts, b.createIndexSQL(IndexDef{
				Name:    b.indexName("idx", []string{col.Name}),
				Columns: []string{col.Name},
			}))
		}
	}

	for _, name := range b.dropColumns {
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", b.q(b.table), b.q(name)))
	}

	for _, idx := range b.indexes {
		stmts = append(stmts, b.createIndexSQL(idx))
	}

	for _, name := range b.dropIndexes {
		stmts = append(stmts, b.dropIndexSQL(name, drv))
	}

	for _, fk := range b.foreignKeys {
		if fk.RefTable == "" {
			continue
		}
		if drv == "sqlite" {
			stmts = append(stmts, fmt.Sprintf("-- sqlite does not support ALTER TABLE ADD CONSTRAINT (%s)", fk.Name))
			continue
		}
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD %s;", b.q(b.table), b.fkConstraintSQL(fk)))
	}

	for _, name := range b.dropForeigns {
		if drv == "sqlite" {
			stmts = append(stmts, fmt.Sprintf("-- sqlite does not support DROP CONSTRAINT (%s)", name))
			continue
		}
		stmts = append(stmts, b.dropForeignSQL(name, drv))
	}

	return strings.Join(stmts, "\n")
}

func (b *Blueprint) dropIndexSQL(name, drv string) string {
	if drv == "mysql" {
		return fmt.Sprintf("DROP INDEX %s ON %s;", b.q(name), b.q(b.table))
	}
	return fmt.Sprintf("DROP INDEX IF EXISTS %s;", b.q(name))
}

func (b *Blueprint) dropForeignSQL(name, drv string) string {
	if drv == "mysql" {
		return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s;", b.q(b.table), b.q(name))
	}
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.q(b.table), b.q(name))
}
