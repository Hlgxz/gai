package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func seedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Run database seeders",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeed()
		},
	}
}

func runSeed() error {
	if err := mustProjectRoot(); err != nil {
		return err
	}

	registry := filepath.Join("database", "seeders", "registry.go")
	if _, err := os.Stat(registry); err != nil {
		fmt.Println("No seeders found. Create one with: gai make seeder User")
		return nil
	}

	if err := writeSeedRunner(); err != nil {
		return err
	}
	fmt.Println("Running seeders...")
	return runGo("run", "./database/seeders/cmd")
}

func writeSeedRunner() error {
	module := detectModule()
	src := formatGoSource(fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/Hlgxz/gai/config"
	"github.com/Hlgxz/gai/database"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/seed"
	"%s/database/seeders"
)

func main() {
	_ = config.LoadEnvFile(".env")
	cfg := config.New()
	if err := cfg.Load("config"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db, err := database.Open(database.Config{
		Driver: cfg.GetString("app.database.driver", "sqlite"),
		DSN:    cfg.GetString("app.database.dsn", "storage/database.db"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed.Run(db, seeders.Seeders...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Seeding complete.")
}
`, module))
	return writeFile("database/seeders/cmd/main.go", src)
}

func ensureSeederRegistry(dir string) error {
	path := filepath.Join(dir, "registry.go")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := `package seeders

import "github.com/Hlgxz/gai/database/seed"

// Seeders collects all seeders registered via init() functions.
var Seeders []seed.Seeder
`
	return os.WriteFile(path, []byte(content), 0o644)
}
