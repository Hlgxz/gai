package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Find returns a single model by primary key.
func Find[T any](db *DB, id any, ctxs ...context.Context) (*T, error) {
	q := Query[T](db).Where("id", "=", id)
	if len(ctxs) > 0 {
		q = q.WithContext(ctxs[0])
	}
	return First[T](q)
}

// FirstOrCreate returns the first matching row, or creates attrs if none exists.
func FirstOrCreate[T any](q *QueryBuilder, attrs *T) (*T, error) {
	found, err := First[T](q)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return found, nil
	}
	return Create[T](q.db, attrs, q.ctx)
}

// UpdateOrCreate finds a row matching the current query; updates it with attrs
// if found, otherwise inserts attrs.
func UpdateOrCreate[T any](q *QueryBuilder, attrs *T) (*T, error) {
	found, err := First[T](q)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return Create[T](q.db, attrs, q.ctx)
	}

	src := reflect.ValueOf(attrs).Elem()
	dst := reflect.ValueOf(found).Elem()
	copyNonZeroFields(dst, src)
	if err := Update[T](q.db, found, q.ctx); err != nil {
		return nil, err
	}
	return found, nil
}

func copyNonZeroFields(dst, src reflect.Value) {
	t := src.Type()
	for i := 0; i < src.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			copyNonZeroFields(dst.Field(i), src.Field(i))
			continue
		}
		sv := src.Field(i)
		if sv.IsZero() {
			continue
		}
		dv := dst.Field(i)
		if dv.CanSet() {
			dv.Set(sv)
		}
	}
}

// Insert bulk-inserts models in a single statement.
func Insert[T any](db *DB, models []T, ctxs ...context.Context) error {
	if len(models) == 0 {
		return nil
	}
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	table := TableName(models[0])
	fields := ParseFields(models[0])
	now := time.Now()

	var cols []string
	var colMeta []FieldInfo
	for _, f := range fields {
		if f.PrimaryKey || f.Relation != "" {
			continue
		}
		cols = append(cols, db.quote(f.Column))
		colMeta = append(colMeta, f)
	}

	var placeholders []string
	var args []any
	idx := 1
	for i := range models {
		v := reflect.ValueOf(&models[i]).Elem()
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		row := make([]string, len(colMeta))
		for j, f := range colMeta {
			val := fieldValue(v, f.Name)
			if f.Column == "created_at" || f.Column == "updated_at" {
				val = now
			}
			row[j] = placeholder(db.DriverName, idx)
			args = append(args, bindArg(val))
			idx++
		}
		placeholders = append(placeholders, "("+strings.Join(row, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		db.quote(table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
	_, err := db.execContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("gai/orm: bulk insert failed: %w", err)
	}
	return nil
}

// ForceDelete permanently removes a record, ignoring soft deletes.
func ForceDelete[T any](db *DB, model *T, ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	if err := callHook(model, "BeforeDelete"); err != nil {
		return err
	}
	table := TableName(*model)
	v := reflect.ValueOf(model).Elem()
	idVal := fieldValue(v, "ID")
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		db.quote(table), db.quote("id"), placeholder(db.DriverName, 1))
	if _, err := db.execContext(ctx, query, idVal); err != nil {
		return fmt.Errorf("gai/orm: force delete failed: %w", err)
	}
	return callHook(model, "AfterDelete")
}

// Restore clears deleted_at on a soft-deleted record.
func Restore[T any](db *DB, model *T, ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	table := TableName(*model)
	v := reflect.ValueOf(model).Elem()
	idVal := fieldValue(v, "ID")
	query := fmt.Sprintf("UPDATE %s SET %s = NULL WHERE %s = %s",
		db.quote(table), db.quote("deleted_at"),
		db.quote("id"), placeholder(db.DriverName, 1))
	if _, err := db.execContext(ctx, query, idVal); err != nil {
		return fmt.Errorf("gai/orm: restore failed: %w", err)
	}
	setFieldValue(v, "DeletedAt", (*time.Time)(nil))
	return nil
}

// Exists reports whether any row matches the query.
func Exists(q *QueryBuilder) (bool, error) {
	n, err := Count(q)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Pluck returns a single column as a slice of T.
func Pluck[T any](q *QueryBuilder, column string) ([]T, error) {
	pq := q.clone()
	pq.selects = []string{safeColumn(column)}
	query, args := pq.buildSelect()
	rows, err := q.db.queryContext(q.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gai/orm: pluck failed: %w", err)
	}
	defer rows.Close()

	var out []T
	for rows.Next() {
		var v T
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Sum returns SUM(column).
func Sum(q *QueryBuilder, column string) (float64, error) {
	return aggregateFloat(q, "SUM", column)
}

// Avg returns AVG(column).
func Avg(q *QueryBuilder, column string) (float64, error) {
	return aggregateFloat(q, "AVG", column)
}

// Max returns MAX(column) as float64.
func Max(q *QueryBuilder, column string) (float64, error) {
	return aggregateFloat(q, "MAX", column)
}

// Min returns MIN(column) as float64.
func Min(q *QueryBuilder, column string) (float64, error) {
	return aggregateFloat(q, "MIN", column)
}

func aggregateFloat(q *QueryBuilder, fn, column string) (float64, error) {
	cq := q.clone()
	cq.selects = []string{fmt.Sprintf("%s(%s) AS agg", fn, safeColumn(column))}
	cq.limitVal = 0
	cq.offsetVal = 0
	query, args := cq.buildSelect()
	var val sql.NullFloat64
	if err := q.db.queryRowContext(q.ctx, query, args...).Scan(&val); err != nil {
		return 0, fmt.Errorf("gai/orm: %s failed: %w", fn, err)
	}
	return val.Float64, nil
}

// Increment adds amount to a numeric column for rows matching the query.
func (q *QueryBuilder) Increment(column string, amount int) error {
	return q.adjust(column, amount)
}

// Decrement subtracts amount from a numeric column for rows matching the query.
func (q *QueryBuilder) Decrement(column string, amount int) error {
	return q.adjust(column, -amount)
}

func (q *QueryBuilder) adjust(column string, amount int) error {
	col := safeColumn(column)
	op := "+"
	n := amount
	if amount < 0 {
		op = "-"
		n = -amount
	}
	query := fmt.Sprintf("UPDATE %s SET %s = %s %s %s",
		q.db.quote(q.table), q.db.quote(col), q.db.quote(col), op, placeholder(q.db.DriverName, 1))
	args := []any{n}

	effective := q.wheres
	if q.softDelete {
		effective = append([]whereClause{{raw: "deleted_at IS NULL", boolean: "AND"}}, effective...)
	}
	if len(effective) > 0 {
		query += " WHERE "
		idx := 2
		var buf strings.Builder
		for i, w := range effective {
			if i > 0 {
				buf.WriteString(" " + w.boolean + " ")
			}
			if w.raw != "" {
				buf.WriteString(w.raw)
				args = append(args, w.rawArgs...)
				idx += len(w.rawArgs)
				continue
			}
			buf.WriteString(fmt.Sprintf("%s %s %s", w.column, w.operator, placeholder(q.db.DriverName, idx)))
			args = append(args, w.value)
			idx++
		}
		query += buf.String()
	}
	_, err := q.db.execContext(q.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("gai/orm: increment/decrement failed: %w", err)
	}
	return nil
}

// ToSQL returns the SELECT SQL and args without executing.
func (q *QueryBuilder) ToSQL() (string, []any) {
	return q.buildSelect()
}

func (q *QueryBuilder) effectiveWheres() []whereClause {
	effective := q.wheres
	if q.softDelete {
		effective = append([]whereClause{{raw: "deleted_at IS NULL", boolean: "AND"}}, effective...)
	} else if q.onlyTrashed {
		effective = append([]whereClause{{raw: "deleted_at IS NOT NULL", boolean: "AND"}}, effective...)
	}
	return effective
}

func (q *QueryBuilder) appendWhere(query string, args []any, idx int) (string, []any) {
	effective := q.effectiveWheres()
	if len(effective) == 0 {
		return query, args
	}
	query += " WHERE "
	var buf strings.Builder
	for i, w := range effective {
		if i > 0 {
			buf.WriteString(" " + w.boolean + " ")
		}
		if w.raw != "" {
			buf.WriteString(w.raw)
			args = append(args, w.rawArgs...)
			idx += len(w.rawArgs)
			continue
		}
		if w.operator == "IN" {
			vals, ok := w.value.([]any)
			if !ok {
				continue
			}
			phs := make([]string, len(vals))
			for j, v := range vals {
				phs[j] = placeholder(q.db.DriverName, idx)
				args = append(args, v)
				idx++
			}
			buf.WriteString(fmt.Sprintf("%s IN (%s)", w.column, strings.Join(phs, ", ")))
			continue
		}
		buf.WriteString(fmt.Sprintf("%s %s %s", w.column, w.operator, placeholder(q.db.DriverName, idx)))
		args = append(args, w.value)
		idx++
	}
	return query + buf.String(), args
}

// UpdateAll sets columns for every row matching the current query.
func (q *QueryBuilder) UpdateAll(values map[string]any) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}
	var sets []string
	var args []any
	idx := 1
	for col, val := range values {
		sets = append(sets, fmt.Sprintf("%s = %s", q.db.quote(safeColumn(col)), placeholder(q.db.DriverName, idx)))
		args = append(args, val)
		idx++
	}
	query := fmt.Sprintf("UPDATE %s SET %s", q.db.quote(q.table), strings.Join(sets, ", "))
	query, args = q.appendWhere(query, args, idx)
	res, err := q.db.execContext(q.ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("gai/orm: update failed: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// DeleteAll deletes (or soft-deletes) every row matching the current query.
func (q *QueryBuilder) DeleteAll() (int64, error) {
	if q.softDelete {
		query := fmt.Sprintf("UPDATE %s SET %s = %s",
			q.db.quote(q.table), q.db.quote("deleted_at"), placeholder(q.db.DriverName, 1))
		args := []any{time.Now()}
		query, args = q.appendWhere(query, args, 2)
		res, err := q.db.execContext(q.ctx, query, args...)
		if err != nil {
			return 0, fmt.Errorf("gai/orm: delete failed: %w", err)
		}
		n, _ := res.RowsAffected()
		return n, nil
	}
	query := fmt.Sprintf("DELETE FROM %s", q.db.quote(q.table))
	query, args := q.appendWhere(query, nil, 1)
	res, err := q.db.execContext(q.ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("gai/orm: delete failed: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
