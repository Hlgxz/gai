package migration_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Hlgxz/gai/database"
	"github.com/Hlgxz/gai/database/driver"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/migration"
)

func sqliteDriver(t *testing.T) driver.Driver {
	t.Helper()
	drv, err := driver.Get("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	return drv
}

func TestBlueprintForeignAndCompositeIndex(t *testing.T) {
	drv := sqliteDriver(t)
	roles := migration.NewBlueprint("roles", drv)
	roles.ID()
	roles.String("name", 50)
	create := roles.ToCreateSQL()
	if !strings.Contains(create, "CREATE TABLE IF NOT EXISTS") {
		t.Fatalf("create sql: %s", create)
	}

	users := migration.NewBlueprint("users", drv)
	users.ID()
	users.String("email", 100).SetUnique()
	users.Integer("role_id")
	users.Index("email", "role_id")
	users.Foreign("role_id").References("id").On("roles").OnDelete("CASCADE")
	sql := users.ToCreateSQL()
	if !strings.Contains(sql, "FOREIGN KEY") {
		t.Fatalf("expected FK in create SQL: %s", sql)
	}
	if !strings.Contains(sql, "CREATE INDEX") && !strings.Contains(sql, "CREATE UNIQUE INDEX") {
		if !strings.Contains(sql, "idx_users_email_role_id") {
			t.Fatalf("expected composite index: %s", sql)
		}
	}
	if !strings.Contains(sql, "idx_users_email_role_id") {
		t.Fatalf("expected composite index name: %s", sql)
	}
}

func TestBlueprintAlterAddAndDrop(t *testing.T) {
	drv := sqliteDriver(t)
	b := migration.NewBlueprint("users", drv)
	b.String("nickname", 80).SetNullable()
	b.DropColumn("old_name")
	b.Index("nickname")
	sql := b.ToAlterSQL()
	if !strings.Contains(sql, "ALTER TABLE") || !strings.Contains(sql, "ADD COLUMN") {
		t.Fatalf("expected ADD COLUMN: %s", sql)
	}
	if !strings.Contains(sql, "DROP COLUMN") {
		t.Fatalf("expected DROP COLUMN: %s", sql)
	}
	if !strings.Contains(sql, "CREATE") || !strings.Contains(sql, "INDEX") {
		t.Fatalf("expected index on alter: %s", sql)
	}
}

func TestAlterForeignSkippedOnSQLite(t *testing.T) {
	drv := sqliteDriver(t)
	b := migration.NewBlueprint("users", drv)
	b.Foreign("role_id").On("roles")
	sql := b.ToAlterSQL()
	if !strings.Contains(sql, "sqlite does not support") {
		t.Fatalf("sqlite alter FK should be skipped: %s", sql)
	}
}

func TestExecuteMigrateAndStatus(t *testing.T) {
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	drv := sqliteDriver(t)

	migs := []migration.Migration{{
		Name: "001_create_items",
		Up: func(d driver.Driver) string {
			b := migration.NewBlueprint("items", d)
			b.ID()
			b.String("title", 40)
			return b.ToCreateSQL()
		},
		Down: func(d driver.Driver) string {
			return migration.NewBlueprint("items", d).ToDropSQL()
		},
	}}

	var buf bytes.Buffer
	if err := migration.ExecuteTo(&buf, db.SQL, drv, migs, "migrate"); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := migration.ExecuteTo(&buf, db.SQL, drv, migs, "status"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Ran") {
		t.Fatalf("status: %s", buf.String())
	}
	if err := migration.ExecuteTo(&buf, db.SQL, drv, migs, "rollback"); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	err = migration.Execute(db.SQL, sqliteDriver(t), nil, "nope")
	if err == nil {
		t.Fatal("expected error")
	}
}
