package orm

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"
)

// executor is satisfied by both *sql.DB and *sql.Tx so queries can run
// inside or outside a transaction without changing call sites.
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// LogFunc is called after each SQL statement when set on DB.
type LogFunc func(query string, args []any, duration time.Duration)

var savepointSeq atomic.Uint64

func (db *DB) executor() executor {
	if db.tx != nil {
		return db.tx
	}
	return db.SQL
}

func (db *DB) log(query string, args []any, start time.Time) {
	if db.Logger != nil {
		db.Logger(query, args, time.Since(start))
	}
}

func (db *DB) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := db.executor().ExecContext(ctx, query, args...)
	db.log(query, args, start)
	return res, err
}

func (db *DB) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.executor().QueryContext(ctx, query, args...)
	db.log(query, args, start)
	return rows, err
}

func (db *DB) queryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := db.executor().QueryRowContext(ctx, query, args...)
	db.log(query, args, start)
	return row
}

// Exec runs a raw SQL statement.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.execContext(ctx, query, args...)
}

// Query runs a raw SQL query and returns rows.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.queryContext(ctx, query, args...)
}

// Ping verifies the database connection is alive.
func (db *DB) Ping(ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	return db.SQL.PingContext(ctx)
}

// Close closes the underlying connection pool.
func (db *DB) Close() error {
	return db.SQL.Close()
}

// InTransaction reports whether this DB handle is bound to an active transaction.
func (db *DB) InTransaction() bool {
	return db.tx != nil
}

// Transaction runs fn inside a database transaction. Nested calls use SAVEPOINT.
// If fn returns an error the transaction (or savepoint) is rolled back.
func (db *DB) Transaction(ctx context.Context, fn func(tx *DB) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if db.tx != nil {
		sp := fmt.Sprintf("sp_%d", savepointSeq.Add(1))
		if _, err := db.tx.ExecContext(ctx, "SAVEPOINT "+sp); err != nil {
			return fmt.Errorf("gai/orm: savepoint: %w", err)
		}
		if err := fn(db); err != nil {
			_, _ = db.tx.ExecContext(ctx, "ROLLBACK TO "+sp)
			return err
		}
		if _, err := db.tx.ExecContext(ctx, "RELEASE "+sp); err != nil {
			return fmt.Errorf("gai/orm: release savepoint: %w", err)
		}
		return nil
	}

	sqlTx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gai/orm: begin tx: %w", err)
	}

	txDB := &DB{
		SQL:        db.SQL,
		tx:         sqlTx,
		DriverName: db.DriverName,
		QuoteIdent: db.QuoteIdent,
		Logger:     db.Logger,
	}

	if err := fn(txDB); err != nil {
		if rbErr := sqlTx.Rollback(); rbErr != nil {
			return fmt.Errorf("gai/orm: rollback after %w: %v", err, rbErr)
		}
		return err
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("gai/orm: commit: %w", err)
	}
	return nil
}

// SetLogger enables SQL statement logging.
func (db *DB) SetLogger(fn LogFunc) {
	db.Logger = fn
}

// ConfigurePool sets connection pool parameters.
func (db *DB) ConfigurePool(maxOpen, maxIdle int, maxLifetime time.Duration) {
	if maxOpen > 0 {
		db.SQL.SetMaxOpenConns(maxOpen)
	}
	if maxIdle > 0 {
		db.SQL.SetMaxIdleConns(maxIdle)
	}
	if maxLifetime > 0 {
		db.SQL.SetConnMaxLifetime(maxLifetime)
	}
}

// ExistsRow reports whether a row exists in table where column = value.
// Used by the HTTP validator unique/exists rules.
func (db *DB) ExistsRow(table, column string, value any) (bool, error) {
	n, err := Count(Table(db, table).Where(column, "=", value))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
