package seed_test

import (
	"testing"

	"github.com/Hlgxz/gai/database"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/orm"
	"github.com/Hlgxz/gai/database/seed"
)

func TestRunSeeders(t *testing.T) {
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	n := 0
	err = seed.Run(db, func(db *orm.DB) error {
		n++
		return nil
	}, func(db *orm.DB) error {
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ran %d seeders", n)
	}
}
