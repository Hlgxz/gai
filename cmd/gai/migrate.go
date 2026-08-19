package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate("migrate")
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "rollback",
			Short: "Rollback the last migration batch",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runMigrate("rollback")
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show migration status",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runMigrate("status")
			},
		},
		&cobra.Command{
			Use:   "fresh",
			Short: "Rollback all migrations and re-run them",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runMigrate("fresh")
			},
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Rollback all migrations",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runMigrate("reset")
			},
		},
	)

	return cmd
}

func runMigrate(command string) error {
	if err := mustProjectRoot(); err != nil {
		return err
	}
	if err := ensureMigrationRegistry("database/migrations"); err != nil {
		return err
	}
	if err := writeMigrateRunner(); err != nil {
		return err
	}
	fmt.Printf("gai migrate %s\n", command)
	return runGo("run", "./database/migrations/cmd", command)
}

func writeMigrateRunner() error {
	module := detectModule()
	src := formatGoSource(fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/Hlgxz/gai/config"
	"github.com/Hlgxz/gai/database"
	"github.com/Hlgxz/gai/database/driver"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/migration"
	"%s/database/migrations"
)

func main() {
	command := "migrate"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

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

	drv, err := driver.Get(db.DriverName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := migration.Execute(db.SQL, drv, migrations.Migrations, command); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, module))
	return writeFile("database/migrations/cmd/main.go", src)
}
