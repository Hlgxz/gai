package migration_test

import (
	"testing"

	"github.com/Hlgxz/gai/database"
	"github.com/Hlgxz/gai/database/driver"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/migration"
)

func TestMigrateRollback(t *testing.T) {
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	drv, _ := driver.Get("sqlite")
	m := migration.NewMigrator(db.SQL, drv)
	m.Add(migration.Migration{
		Name: "001_create_notes",
		Up: func(d driver.Driver) string {
			b := migration.NewBlueprint("notes", d)
			b.ID()
			b.String("title", 100)
			return b.ToCreateSQL()
		},
		Down: func(d driver.Driver) string {
			return migration.NewBlueprint("notes", d).ToDropSQL()
		},
	})
	if err := m.Migrate(); err != nil {
		t.Fatal(err)
	}
	st, _ := m.Status()
	if len(st) != 1 || !st[0].Ran {
		t.Fatalf("status %+v", st)
	}
	if err := m.Rollback(); err != nil {
		t.Fatal(err)
	}
}
