package migration

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/Hlgxz/gai/database/driver"
)

// Execute runs a migrator command against the given migration list.
// command is one of: migrate (default), rollback, status, fresh, reset.
func Execute(db *sql.DB, drv driver.Driver, migrations []Migration, command string) error {
	return ExecuteTo(os.Stdout, db, drv, migrations, command)
}

// ExecuteTo is like Execute but writes status output to w.
func ExecuteTo(w io.Writer, db *sql.DB, drv driver.Driver, migrations []Migration, command string) error {
	m := NewMigrator(db, drv)
	for _, mig := range migrations {
		m.Add(mig)
	}

	switch command {
	case "", "migrate", "up":
		fmt.Fprintln(w, "Running migrations...")
		if err := m.Migrate(); err != nil {
			return err
		}
		fmt.Fprintln(w, "Migrations complete.")
		return nil
	case "rollback":
		fmt.Fprintln(w, "Rolling back last migration batch...")
		if err := m.Rollback(); err != nil {
			return err
		}
		fmt.Fprintln(w, "Rollback complete.")
		return nil
	case "status":
		return printStatus(w, m)
	case "fresh":
		fmt.Fprintln(w, "Fresh: rollback all and re-run...")
		if err := m.Fresh(); err != nil {
			return err
		}
		fmt.Fprintln(w, "Fresh complete.")
		return nil
	case "reset":
		fmt.Fprintln(w, "Resetting all migrations...")
		if err := m.Reset(); err != nil {
			return err
		}
		fmt.Fprintln(w, "Reset complete.")
		return nil
	default:
		return fmt.Errorf("gai/migration: unknown command %q", command)
	}
}

func printStatus(w io.Writer, m *Migrator) error {
	statuses, err := m.Status()
	if err != nil {
		return err
	}
	if len(statuses) == 0 {
		fmt.Fprintln(w, "No migrations registered.")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tMIGRATION")
	for _, st := range statuses {
		mark := "Pending"
		if st.Ran {
			mark = "Ran"
		}
		fmt.Fprintf(tw, "%s\t%s\n", mark, st.Name)
	}
	return tw.Flush()
}
