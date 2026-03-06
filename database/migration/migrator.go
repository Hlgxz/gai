package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/Hlgxz/gai/database/driver"
)

// Migration represents a single database migration with Up and Down methods.
type Migration struct {
	Name string
	Up   func(drv driver.Driver) string
	Down func(drv driver.Driver) string
}

// Migrator manages the execution and rollback of migrations, tracking state
// in a "migrations" table similar to Laravel's migration system.
type Migrator struct {
	db         *sql.DB
	driver     driver.Driver
	migrations []Migration
}

// NewMigrator creates a Migrator for the given database connection.
func NewMigrator(db *sql.DB, drv driver.Driver) *Migrator {
	return &Migrator{db: db, driver: drv}
}

// Add registers a migration.
func (m *Migrator) Add(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

// Migrate runs all pending migrations in order.
func (m *Migrator) Migrate() error {
	if err := m.ensureTable(); err != nil {
		return err
	}

	ran, err := m.ranMigrations()
	if err != nil {
		return err
	}
	ranSet := make(map[string]bool, len(ran))
	for _, name := range ran {
		ranSet[name] = true
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Name < m.migrations[j].Name
	})

	batch, _ := m.lastBatch()
	batch++

	for _, mig := range m.migrations {
		if ranSet[mig.Name] {
			continue
		}

		sqlStr := mig.Up(m.driver)
		slog.Info("migrating", "name", mig.Name)

		if _, err := m.db.Exec(sqlStr); err != nil {
			return fmt.Errorf("gai/migration: failed to run %s: %w", mig.Name, err)
		}

		if err := m.record(mig.Name, batch); err != nil {
			return err
		}
		slog.Info("migrated", "name", mig.Name)
	}

	return nil
}

// Rollback undoes the last batch of migrations.
func (m *Migrator) Rollback() error {
	if err := m.ensureTable(); err != nil {
		return err
	}

	batch, err := m.lastBatch()
	if err != nil {
		return err
	}
	if batch == 0 {
		slog.Info("nothing to rollback")
		return nil
	}

	names, err := m.batchMigrations(batch)
	if err != nil {
		return err
	}

	migMap := make(map[string]Migration)
	for _, mig := range m.migrations {
		migMap[mig.Name] = mig
	}

	// Rollback in reverse order.
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		mig, ok := migMap[name]
		if !ok {
			slog.Warn("migration not found in code", "name", name)
			continue
		}

		sqlStr := mig.Down(m.driver)
		slog.Info("rolling back", "name", name)

		if _, err := m.db.Exec(sqlStr); err != nil {
			return fmt.Errorf("gai/migration: failed to rollback %s: %w", name, err)
		}

		if err := m.removeRecord(name); err != nil {
			return err
		}
		slog.Info("rolled back", "name", name)
	}

	return nil
}

// Status returns a list of migration names and whether they have been run.
func (m *Migrator) Status() ([]MigrationStatus, error) {
	if err := m.ensureTable(); err != nil {
		return nil, err
	}

	ran, err := m.ranMigrations()
	if err != nil {
		return nil, err
	}
	ranSet := make(map[string]bool, len(ran))
	for _, name := range ran {
		ranSet[name] = true
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Name < m.migrations[j].Name
	})

	result := make([]MigrationStatus, len(m.migrations))
	for i, mig := range m.migrations {
		result[i] = MigrationStatus{
			Name: mig.Name,
			Ran:  ranSet[mig.Name],
		}
	}
	return result, nil
}

// MigrationStatus represents the run state of a migration.
type MigrationStatus struct {
	Name string
	Ran  bool
}

// ---------------------------------------------------------- Internal

func (m *Migrator) ensureTable() error {
	sql := `CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY,
		migration VARCHAR(255) NOT NULL,
		batch INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if m.driver.Name() == "mysql" {
		sql = `CREATE TABLE IF NOT EXISTS migrations (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	}
	_, err := m.db.Exec(sql)
	return err
}

func (m *Migrator) ranMigrations() ([]string, error) {
	rows, err := m.db.Query("SELECT migration FROM migrations ORDER BY migration")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (m *Migrator) lastBatch() (int, error) {
	var batch sql.NullInt64
	err := m.db.QueryRow("SELECT MAX(batch) FROM migrations").Scan(&batch)
	if err != nil {
		return 0, err
	}
	if !batch.Valid {
		return 0, nil
	}
	return int(batch.Int64), nil
}

func (m *Migrator) batchMigrations(batch int) ([]string, error) {
	rows, err := m.db.Query("SELECT migration FROM migrations WHERE batch = ? ORDER BY migration", batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (m *Migrator) record(name string, batch int) error {
	_, err := m.db.Exec("INSERT INTO migrations (migration, batch, created_at) VALUES (?, ?, ?)",
		name, batch, time.Now())
	return err
}

func (m *Migrator) removeRecord(name string) error {
	_, err := m.db.Exec("DELETE FROM migrations WHERE migration = ?", name)
	return err
}
