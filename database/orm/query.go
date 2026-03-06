package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Hlgxz/gai/support"
)

// DB wraps *sql.DB with driver metadata.
type DB struct {
	SQL        *sql.DB
	DriverName string
	QuoteIdent func(name string) string
}

func (db *DB) quote(name string) string {
	if db.QuoteIdent != nil {
		return db.QuoteIdent(name)
	}
	return name
}

// Pagination holds a paginated result set.
type Pagination[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// QueryBuilder provides a fluent, chainable interface for building SQL queries,
// inspired by Laravel's Eloquent query builder.
type QueryBuilder struct {
	db         *DB
	ctx        context.Context
	table      string
	selects    []string
	wheres     []whereClause
	orders     []orderClause
	limitVal   int
	offsetVal  int
	groupBy    []string
	having     []whereClause
	softDelete bool
	modelType  reflect.Type
}

type whereClause struct {
	column   string
	operator string
	value    any
	boolean  string // "AND" or "OR"
	raw      string
	rawArgs  []any
}

type orderClause struct {
	column    string
	direction string
}

// Query creates a new QueryBuilder for the given model type. T must be a
// struct that embeds orm.Model.
func Query[T any](db *DB) *QueryBuilder {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	qb := &QueryBuilder{
		db:        db,
		ctx:       context.Background(),
		table:     TableName(zero),
		modelType: t,
	}

	// Auto-apply soft delete scope if model has DeletedAt field.
	fields := ParseFields(zero)
	for _, f := range fields {
		if f.SoftDelete {
			qb.softDelete = true
			break
		}
	}

	return qb
}

// Table creates a raw QueryBuilder for a table (no model binding).
func Table(db *DB, table string) *QueryBuilder {
	return &QueryBuilder{db: db, ctx: context.Background(), table: table}
}

// WithContext sets the context for all SQL operations in this query.
func (q *QueryBuilder) WithContext(ctx context.Context) *QueryBuilder {
	q.ctx = ctx
	return q
}

// ---------------------------------------------------------- Chainable

// Select specifies which columns to retrieve.
func (q *QueryBuilder) Select(columns ...string) *QueryBuilder {
	q.selects = append(q.selects, columns...)
	return q
}

// Where adds an AND condition.
func (q *QueryBuilder) Where(column, operator string, value any) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{column: safeColumn(column), operator: sanitizeOperator(operator), value: value, boolean: "AND"})
	return q
}

// OrWhere adds an OR condition.
func (q *QueryBuilder) OrWhere(column, operator string, value any) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{column: safeColumn(column), operator: sanitizeOperator(operator), value: value, boolean: "OR"})
	return q
}

// WhereNull adds an IS NULL condition.
func (q *QueryBuilder) WhereNull(column string) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{raw: safeColumn(column) + " IS NULL", boolean: "AND"})
	return q
}

// WhereNotNull adds an IS NOT NULL condition.
func (q *QueryBuilder) WhereNotNull(column string) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{raw: safeColumn(column) + " IS NOT NULL", boolean: "AND"})
	return q
}

// WhereIn adds a WHERE column IN (...) condition.
func (q *QueryBuilder) WhereIn(column string, values ...any) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{column: safeColumn(column), operator: "IN", value: values, boolean: "AND"})
	return q
}

// WhereRaw adds a raw WHERE clause with optional parameterized args.
func (q *QueryBuilder) WhereRaw(raw string, args ...any) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{raw: raw, boolean: "AND", rawArgs: args})
	return q
}

// OrderBy adds an ORDER BY clause.
func (q *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	dir := strings.ToUpper(direction)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	q.orders = append(q.orders, orderClause{column: safeColumn(column), direction: dir})
	return q
}

// Limit sets the maximum number of rows.
func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	q.limitVal = n
	return q
}

// Offset sets the row offset.
func (q *QueryBuilder) Offset(n int) *QueryBuilder {
	q.offsetVal = n
	return q
}

// GroupBy adds GROUP BY columns.
func (q *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	for _, col := range columns {
		q.groupBy = append(q.groupBy, safeColumn(col))
	}
	return q
}

// Having adds a HAVING clause.
func (q *QueryBuilder) Having(column, operator string, value any) *QueryBuilder {
	q.having = append(q.having, whereClause{column: safeColumn(column), operator: sanitizeOperator(operator), value: value, boolean: "AND"})
	return q
}

// Scope applies a reusable Scope function.
func (q *QueryBuilder) Scope(scopes ...Scope) *QueryBuilder {
	for _, s := range scopes {
		q = s(q)
	}
	return q
}

// WithTrashed disables the automatic soft-delete filter for this query.
func (q *QueryBuilder) WithTrashed() *QueryBuilder {
	q.softDelete = false
	return q
}

// clone returns a shallow copy of the QueryBuilder so that terminal
// operations like Count and Paginate do not mutate the caller's query.
func (q *QueryBuilder) clone() *QueryBuilder {
	c := *q
	c.selects = append([]string(nil), q.selects...)
	c.wheres = append([]whereClause(nil), q.wheres...)
	c.orders = append([]orderClause(nil), q.orders...)
	c.groupBy = append([]string(nil), q.groupBy...)
	c.having = append([]whereClause(nil), q.having...)
	return &c
}

// ---------------------------------------------------------- Terminal operations

// Get executes the query and returns all matching rows scanned into T.
func Get[T any](q *QueryBuilder) ([]T, error) {
	query, args := q.buildSelect()
	rows, err := q.db.SQL.QueryContext(q.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gai/orm: query failed: %w", err)
	}
	defer rows.Close()
	return scanRows[T](rows)
}

// First returns the first matching row without mutating the original query.
func First[T any](q *QueryBuilder) (*T, error) {
	fq := q.clone()
	fq.limitVal = 1
	items, err := Get[T](fq)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// Count returns the number of matching rows without mutating the original query.
func Count(q *QueryBuilder) (int, error) {
	cq := q.clone()
	cq.selects = []string{"COUNT(*) AS cnt"}
	cq.limitVal = 0
	cq.offsetVal = 0
	query, args := cq.buildSelect()

	var count int
	if err := q.db.SQL.QueryRowContext(q.ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("gai/orm: count failed: %w", err)
	}
	return count, nil
}

// Paginate returns a paginated result. Page is 1-based.
// The original QueryBuilder is not mutated.
func Paginate[T any](q *QueryBuilder, page, perPage int) (*Pagination[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	total, err := Count(q)
	if err != nil {
		return nil, err
	}

	pq := q.clone()
	pq.limitVal = perPage
	pq.offsetVal = (page - 1) * perPage

	items, err := Get[T](pq)
	if err != nil {
		return nil, err
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	return &Pagination[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// ---------------------------------------------------------- CRUD helpers

// Create inserts a new record and returns it with the generated ID.
func Create[T any](db *DB, model *T, ctxs ...context.Context) (*T, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	table := TableName(*model)
	fields := ParseFields(*model)

	var cols []string
	var placeholders []string
	var args []any
	v := reflect.ValueOf(model).Elem()
	idx := 1

	now := time.Now()

	for _, f := range fields {
		if f.PrimaryKey || f.Relation != "" {
			continue
		}
		val := fieldValue(v, f.Name)
		if f.Column == "created_at" || f.Column == "updated_at" {
			val = now
		}
		cols = append(cols, db.quote(f.Column))
		placeholders = append(placeholders, placeholder(db.DriverName, idx))
		args = append(args, val)
		idx++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		db.quote(table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	var id int64
	if db.DriverName == "postgres" {
		query += " RETURNING id"
		if err := db.SQL.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			return nil, fmt.Errorf("gai/orm: insert failed: %w", err)
		}
	} else {
		result, err := db.SQL.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("gai/orm: insert failed: %w", err)
		}
		id, _ = result.LastInsertId()
	}

	setFieldValue(v, "ID", uint64(id))
	setFieldValue(v, "CreatedAt", now)
	setFieldValue(v, "UpdatedAt", now)

	return model, nil
}

// Update saves changes to an existing record (identified by ID).
func Update[T any](db *DB, model *T, ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	table := TableName(*model)
	fields := ParseFields(*model)
	v := reflect.ValueOf(model).Elem()

	var sets []string
	var args []any
	idx := 1
	var idVal any

	now := time.Now()

	for _, f := range fields {
		if f.Relation != "" {
			continue
		}
		if f.PrimaryKey {
			idVal = fieldValue(v, f.Name)
			continue
		}
		val := fieldValue(v, f.Name)
		if f.Column == "updated_at" {
			val = now
		}
		sets = append(sets, fmt.Sprintf("%s = %s", db.quote(f.Column), placeholder(db.DriverName, idx)))
		args = append(args, val)
		idx++
	}

	args = append(args, idVal)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		db.quote(table),
		strings.Join(sets, ", "),
		db.quote("id"),
		placeholder(db.DriverName, idx),
	)

	_, err := db.SQL.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("gai/orm: update failed: %w", err)
	}
	setFieldValue(v, "UpdatedAt", now)
	return nil
}

// Delete removes a record. If the model supports soft deletes, it sets
// deleted_at instead of actually removing the row.
func Delete[T any](db *DB, model *T, ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	table := TableName(*model)
	fields := ParseFields(*model)
	v := reflect.ValueOf(model).Elem()

	hasSoftDelete := false
	for _, f := range fields {
		if f.SoftDelete {
			hasSoftDelete = true
			break
		}
	}

	idVal := fieldValue(v, "ID")

	if hasSoftDelete {
		now := time.Now()
		query := fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = %s",
			db.quote(table), db.quote("deleted_at"), placeholder(db.DriverName, 1),
			db.quote("id"), placeholder(db.DriverName, 2))
		if _, err := db.SQL.ExecContext(ctx, query, now, idVal); err != nil {
			return fmt.Errorf("gai/orm: soft delete failed: %w", err)
		}
		return nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		db.quote(table), db.quote("id"), placeholder(db.DriverName, 1))
	if _, err := db.SQL.ExecContext(ctx, query, idVal); err != nil {
		return fmt.Errorf("gai/orm: delete failed: %w", err)
	}
	return nil
}

// ---------------------------------------------------------- SQL Building

func (q *QueryBuilder) buildSelect() (string, []any) {
	var buf strings.Builder
	var args []any
	idx := 1

	sel := "*"
	if len(q.selects) > 0 {
		sel = strings.Join(q.selects, ", ")
	}
	buf.WriteString("SELECT ")
	buf.WriteString(sel)
	buf.WriteString(" FROM ")
	buf.WriteString(q.db.quote(q.table))

	// Inject soft-delete filter.
	effectiveWheres := q.wheres
	if q.softDelete {
		effectiveWheres = append([]whereClause{{raw: "deleted_at IS NULL", boolean: "AND"}}, effectiveWheres...)
	}

	if len(effectiveWheres) > 0 {
		buf.WriteString(" WHERE ")
		for i, w := range effectiveWheres {
			if i > 0 {
				buf.WriteString(" " + w.boolean + " ")
			}
			if w.raw != "" {
				buf.WriteString(w.raw)
				args = append(args, w.rawArgs...)
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
			} else {
				buf.WriteString(fmt.Sprintf("%s %s %s", w.column, w.operator, placeholder(q.db.DriverName, idx)))
				args = append(args, w.value)
				idx++
			}
		}
	}

	if len(q.groupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		buf.WriteString(strings.Join(q.groupBy, ", "))
	}

	if len(q.having) > 0 {
		buf.WriteString(" HAVING ")
		for i, h := range q.having {
			if i > 0 {
				buf.WriteString(" " + h.boolean + " ")
			}
			buf.WriteString(fmt.Sprintf("%s %s %s", h.column, h.operator, placeholder(q.db.DriverName, idx)))
			args = append(args, h.value)
			idx++
		}
	}

	if len(q.orders) > 0 {
		buf.WriteString(" ORDER BY ")
		parts := make([]string, len(q.orders))
		for i, o := range q.orders {
			parts[i] = o.column + " " + o.direction
		}
		buf.WriteString(strings.Join(parts, ", "))
	}

	if q.limitVal > 0 {
		buf.WriteString(fmt.Sprintf(" LIMIT %d", q.limitVal))
	}
	if q.offsetVal > 0 {
		buf.WriteString(fmt.Sprintf(" OFFSET %d", q.offsetVal))
	}

	return buf.String(), args
}

// ---------------------------------------------------------- Reflection helpers

func scanRows[T any](rows *sql.Rows) ([]T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []T
	for rows.Next() {
		var item T
		v := reflect.ValueOf(&item).Elem()
		dest := mapColumnsToFields(v, cols)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("gai/orm: scan failed: %w", err)
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func mapColumnsToFields(v reflect.Value, cols []string) []any {
	t := v.Type()
	fieldMap := buildFieldMap(t, v)
	dest := make([]any, len(cols))
	for i, col := range cols {
		if ptr, ok := fieldMap[col]; ok {
			dest[i] = ptr
		} else {
			var discard any
			dest[i] = &discard
		}
	}
	return dest
}

func buildFieldMap(t reflect.Type, v reflect.Value) map[string]any {
	m := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			sub := buildFieldMap(f.Type, v.Field(i))
			for k, ptr := range sub {
				m[k] = ptr
			}
			continue
		}
		tag := f.Tag.Get("gai")
		col := parseColumn(tag, f.Name)
		m[col] = v.Field(i).Addr().Interface()
	}
	return m
}

func parseColumn(tag, fieldName string) string {
	if tag == "" {
		return support.Snake(fieldName)
	}
	for _, part := range strings.Split(tag, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if kv[0] == "column" && len(kv) == 2 {
			return kv[1]
		}
	}
	return support.Snake(fieldName)
}

func fieldValue(v reflect.Value, name string) any {
	fv := v.FieldByName(name)
	if !fv.IsValid() {
		// Search embedded structs.
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Anonymous && v.Field(i).Kind() == reflect.Struct {
				if inner := v.Field(i).FieldByName(name); inner.IsValid() {
					return inner.Interface()
				}
			}
		}
		return nil
	}
	return fv.Interface()
}

func setFieldValue(v reflect.Value, name string, val any) {
	fv := v.FieldByName(name)
	if !fv.IsValid() {
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Anonymous && v.Field(i).Kind() == reflect.Struct {
				if inner := v.Field(i).FieldByName(name); inner.IsValid() {
					inner.Set(reflect.ValueOf(val))
					return
				}
			}
		}
		return
	}
	fv.Set(reflect.ValueOf(val))
}

func placeholder(driver string, n int) string {
	if driver == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

var validOperators = map[string]bool{
	"=": true, "!=": true, "<>": true,
	">": true, "<": true, ">=": true, "<=": true,
	"LIKE": true, "like": true, "NOT LIKE": true, "not like": true,
	"IN": true, "in": true,
	"IS": true, "is": true, "IS NOT": true, "is not": true,
}

func sanitizeOperator(op string) string {
	if validOperators[op] {
		return op
	}
	return "="
}

func safeColumn(col string) string {
	var buf strings.Builder
	for _, r := range col {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.' {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
