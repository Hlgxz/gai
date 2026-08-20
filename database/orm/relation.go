package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Hlgxz/gai/support"
)

// With eagerly loads a named relation on the given slice of models.
// Nested paths like "Posts.Comments" are supported.
func With[T any](db *DB, models []T, relation string, ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	parts := strings.Split(relation, ".")
	return withPath(db, ctx, models, parts)
}

func withPath[T any](db *DB, ctx context.Context, models []T, parts []string) error {
	if len(models) == 0 || len(parts) == 0 {
		return nil
	}
	if err := eagerLoad[T](db, ctx, models, parts[0]); err != nil {
		return err
	}
	if len(parts) == 1 {
		return nil
	}
	children := childPointers(models, parts[0])
	return withPath[any](db, ctx, children, parts[1:])
}

func childPointers[T any](models []T, field string) []any {
	var out []any
	for i := range models {
		v := derefModel(&models[i])
		fv := v.FieldByName(field)
		if !fv.IsValid() {
			continue
		}
		switch fv.Kind() {
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				el := fv.Index(j)
				if el.Kind() == reflect.Struct && el.CanAddr() {
					out = append(out, el.Addr().Interface())
				} else if el.Kind() == reflect.Ptr && !el.IsNil() {
					out = append(out, el.Interface())
				}
			}
		case reflect.Ptr:
			if !fv.IsNil() {
				out = append(out, fv.Interface())
			}
		case reflect.Struct:
			if fv.CanAddr() {
				out = append(out, fv.Addr().Interface())
			}
		}
	}
	return out
}

func derefModel(ptr any) reflect.Value {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func modelElem[T any](models []T, i int) reflect.Value {
	v := reflect.ValueOf(&models[i]).Elem()
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func parentNameOf[T any](models []T) string {
	if len(models) == 0 {
		return ""
	}
	t := reflect.TypeOf(models[0])
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	return t.Name()
}

func eagerLoad[T any](db *DB, ctx context.Context, models []T, relation string) error {
	if len(models) == 0 {
		return nil
	}

	t := reflect.TypeOf(models[0])
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}

	field, ok := t.FieldByName(relation)
	if !ok {
		return fmt.Errorf("gai/orm: relation %q not found on %s", relation, t.Name())
	}

	tag := field.Tag.Get("gai")
	relType, meta := parseRelationMeta(tag)
	if relType == "" {
		relType = "hasMany"
	}

	switch relType {
	case "hasMany":
		return loadHasMany(db, ctx, models, relation, field, meta)
	case "hasOne":
		return loadHasOne(db, ctx, models, relation, field, meta)
	case "belongsTo":
		return loadBelongsTo(db, ctx, models, relation, field, meta)
	case "belongsToMany":
		return loadBelongsToMany(db, ctx, models, relation, field, meta)
	}

	return fmt.Errorf("gai/orm: unsupported relation type %q", relType)
}

type relationMeta struct {
	model     string
	pivot     string
	fk        string
	relatedFK string
}

func loadHasMany[T any](db *DB, ctx context.Context, models []T, relation string, field reflect.StructField, meta relationMeta) error {
	ids := make([]any, len(models))
	for i := range models {
		ids[i] = fieldValue(modelElem(models, i), "ID")
	}

	childType := field.Type.Elem()
	childTable := strings.ToLower(support.Plural(support.Snake(childType.Name())))
	fk := meta.fk
	if fk == "" {
		fk = support.Snake(parentNameOf(models)) + "_id"
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = placeholder(db.DriverName, i+1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		db.quote(childTable), db.quote(fk), strings.Join(placeholders, ", "))

	rows, err := db.queryContext(ctx, query, ids...)
	if err != nil {
		return fmt.Errorf("gai/orm: hasMany query failed: %w", err)
	}
	defer rows.Close()

	childItems, err := scanRowsDynamic(rows, childType)
	if err != nil {
		return err
	}

	grouped := make(map[any][]reflect.Value)
	for _, child := range childItems {
		fkVal := fieldValue(child, support.Camel(fk))
		grouped[fkVal] = append(grouped[fkVal], reflect.ValueOf(child.Interface()))
	}

	for i := range models {
		v := modelElem(models, i)
		parentID := fieldValue(v, "ID")
		children := grouped[parentID]
		sliceVal := reflect.MakeSlice(field.Type, len(children), len(children))
		for j, c := range children {
			sliceVal.Index(j).Set(c)
		}
		v.FieldByName(relation).Set(sliceVal)
	}

	return nil
}

func loadHasOne[T any](db *DB, ctx context.Context, models []T, relation string, field reflect.StructField, meta relationMeta) error {
	ids := make([]any, len(models))
	for i := range models {
		ids[i] = fieldValue(modelElem(models, i), "ID")
	}

	relType := field.Type
	isPtr := relType.Kind() == reflect.Ptr
	if isPtr {
		relType = relType.Elem()
	}

	childTable := strings.ToLower(support.Plural(support.Snake(relType.Name())))
	fk := meta.fk
	if fk == "" {
		fk = support.Snake(parentNameOf(models)) + "_id"
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = placeholder(db.DriverName, i+1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		db.quote(childTable), db.quote(fk), strings.Join(placeholders, ", "))

	rows, err := db.queryContext(ctx, query, ids...)
	if err != nil {
		return fmt.Errorf("gai/orm: hasOne query failed: %w", err)
	}
	defer rows.Close()

	childItems, err := scanRowsDynamic(rows, relType)
	if err != nil {
		return err
	}

	byFK := make(map[any]reflect.Value)
	for _, child := range childItems {
		fkVal := fieldValue(child, support.Camel(fk))
		byFK[fkVal] = reflect.ValueOf(child.Interface())
	}

	for i := range models {
		v := modelElem(models, i)
		parentID := fieldValue(v, "ID")
		rel, ok := byFK[parentID]
		if !ok {
			continue
		}
		fld := v.FieldByName(relation)
		if isPtr {
			ptr := reflect.New(relType)
			ptr.Elem().Set(rel)
			fld.Set(ptr)
		} else {
			fld.Set(rel)
		}
	}

	return nil
}

func loadBelongsTo[T any](db *DB, ctx context.Context, models []T, relation string, field reflect.StructField, meta relationMeta) error {
	relType := field.Type
	if relType.Kind() == reflect.Ptr {
		relType = relType.Elem()
	}

	fk := meta.fk
	if fk == "" {
		fk = support.Snake(relation) + "_id"
	}
	relTable := strings.ToLower(support.Plural(support.Snake(relType.Name())))

	ids := make([]any, 0, len(models))
	for i := range models {
		v := modelElem(models, i)
		id := fieldValue(v, support.Camel(fk))
		if id != nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = placeholder(db.DriverName, i+1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		db.quote(relTable), db.quote("id"), strings.Join(placeholders, ", "))

	rows, err := db.queryContext(ctx, query, ids...)
	if err != nil {
		return fmt.Errorf("gai/orm: belongsTo query failed: %w", err)
	}
	defer rows.Close()

	relItems, err := scanRowsDynamic(rows, relType)
	if err != nil {
		return err
	}

	byID := make(map[any]reflect.Value)
	for _, item := range relItems {
		id := fieldValue(item, "ID")
		byID[id] = reflect.ValueOf(item.Interface())
	}

	for i := range models {
		v := modelElem(models, i)
		fkVal := fieldValue(v, support.Camel(fk))
		if rel, ok := byID[fkVal]; ok {
			fld := v.FieldByName(relation)
			if fld.Kind() == reflect.Ptr {
				ptr := reflect.New(relType)
				ptr.Elem().Set(rel)
				fld.Set(ptr)
			} else {
				fld.Set(rel)
			}
		}
	}

	return nil
}

func loadBelongsToMany[T any](db *DB, ctx context.Context, models []T, relation string, field reflect.StructField, meta relationMeta) error {
	parentName := parentNameOf(models)

	childType := field.Type.Elem()
	childName := childType.Name()
	childTable := strings.ToLower(support.Plural(support.Snake(childName)))

	pivot := meta.pivot
	if pivot == "" {
		a, b := strings.ToLower(support.Snake(parentName)), strings.ToLower(support.Snake(childName))
		if a > b {
			a, b = b, a
		}
		pivot = a + "_" + b
	}
	fk := meta.fk
	if fk == "" {
		fk = support.Snake(parentName) + "_id"
	}
	relatedFK := meta.relatedFK
	if relatedFK == "" {
		relatedFK = support.Snake(childName) + "_id"
	}

	ids := make([]any, len(models))
	for i := range models {
		ids[i] = fieldValue(modelElem(models, i), "ID")
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = placeholder(db.DriverName, i+1)
	}

	pivotQuery := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IN (%s)",
		db.quote(fk), db.quote(relatedFK), db.quote(pivot), db.quote(fk), strings.Join(placeholders, ", "))

	rows, err := db.queryContext(ctx, pivotQuery, ids...)
	if err != nil {
		return fmt.Errorf("gai/orm: belongsToMany pivot query failed: %w", err)
	}

	type pair struct{ parent, related any }
	var pairs []pair
	relatedIDs := make([]any, 0)
	seen := map[any]struct{}{}
	for rows.Next() {
		var parentID, relatedID any
		if err := rows.Scan(&parentID, &relatedID); err != nil {
			rows.Close()
			return err
		}
		pairs = append(pairs, pair{parentID, relatedID})
		if _, ok := seen[relatedID]; !ok {
			seen[relatedID] = struct{}{}
			relatedIDs = append(relatedIDs, relatedID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if len(relatedIDs) == 0 {
		return nil
	}

	ph2 := make([]string, len(relatedIDs))
	for i := range relatedIDs {
		ph2[i] = placeholder(db.DriverName, i+1)
	}
	relQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		db.quote(childTable), db.quote("id"), strings.Join(ph2, ", "))

	relRows, err := db.queryContext(ctx, relQuery, relatedIDs...)
	if err != nil {
		return fmt.Errorf("gai/orm: belongsToMany query failed: %w", err)
	}
	defer relRows.Close()

	relItems, err := scanRowsDynamic(relRows, childType)
	if err != nil {
		return err
	}

	byID := make(map[any]reflect.Value)
	for _, item := range relItems {
		id := fieldValue(item, "ID")
		byID[id] = reflect.ValueOf(item.Interface())
	}

	grouped := make(map[any][]reflect.Value)
	for _, p := range pairs {
		if rel, ok := byID[p.related]; ok {
			grouped[p.parent] = append(grouped[p.parent], rel)
		}
	}

	for i := range models {
		v := modelElem(models, i)
		parentID := fieldValue(v, "ID")
		children := grouped[parentID]
		sliceVal := reflect.MakeSlice(field.Type, len(children), len(children))
		for j, c := range children {
			sliceVal.Index(j).Set(c)
		}
		v.FieldByName(relation).Set(sliceVal)
	}

	return nil
}

func scanRowsDynamic(rows *sql.Rows, t reflect.Type) ([]reflect.Value, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var results []reflect.Value
	for rows.Next() {
		ptr := reflect.New(t)
		v := ptr.Elem()
		dest := mapColumnsToFields(v, cols)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("gai/orm: scan failed: %w", err)
		}
		results = append(results, v)
	}
	return results, rows.Err()
}

func parseRelationMeta(tag string) (string, relationMeta) {
	var relType string
	var meta relationMeta
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, ":", 2)
		key := kv[0]
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}
		switch key {
		case "hasMany", "hasOne", "belongsTo", "belongsToMany":
			relType = key
			meta.model = val
		case "pivot":
			meta.pivot = val
		case "fk":
			meta.fk = val
		case "relatedFk":
			meta.relatedFK = val
		}
	}
	return relType, meta
}
