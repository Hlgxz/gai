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
			fmt.Println("Running migrations...")
			fmt.Println("Note: In your application, register migrations and call migrator.Migrate().")
			fmt.Println("The CLI will detect your database config from config/app.yaml and .env")
			return nil
		},
	}

	cmd.AddCommand(
		migrateRollbackCmd(),
		migrateStatusCmd(),
	)

	return cmd
}

func migrateRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last migration batch",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Rolling back last migration batch...")
			fmt.Println("Note: Wire up migrator.Rollback() in your application.")
			return nil
		},
	}
}

func migrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Migration status:")
			fmt.Println("Note: Wire up migrator.Status() in your application.")
			return nil
		},
	}
}
