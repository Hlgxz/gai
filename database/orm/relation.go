package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Hlgxz/gai/support"
)

// With eagerly loads a named relation on the given slice of models.
// It performs a separate query (N+1 avoided via batch loading).
//
// Usage:
//
//	users, _ := Get[User](qb)
//	orm.With(db, users, "Posts")
func With[T any](db *DB, models []T, relation string) error {
	if len(models) == 0 {
		return nil
	}

	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Find the relation field.
	field, ok := t.FieldByName(relation)
	if !ok {
		return fmt.Errorf("gai/orm: relation %q not found on %s", relation, t.Name())
	}

	tag := field.Tag.Get("gai")
	relType, _ := parseRelationTag(tag)
	if relType == "" {
		relType = "hasMany"
	}

	switch relType {
	case "hasMany":
		return loadHasMany(db, models, relation, field)
	case "belongsTo":
		return loadBelongsTo(db, models, relation, field)
	}

	return fmt.Errorf("gai/orm: unsupported relation type %q", relType)
}

func loadHasMany[T any](db *DB, models []T, relation string, field reflect.StructField) error {
	var zero T
	parentTable := TableName(zero)
	_ = parentTable

	// Collect parent IDs.
	ids := make([]any, len(models))
	for i, m := range models {
		v := reflect.ValueOf(m)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		ids[i] = fieldValue(v, "ID")
	}

	// Determine child table and foreign key.
	childType := field.Type.Elem()
	childTable := strings.ToLower(support.Plural(support.Snake(childType.Name())))
	fk := support.Snake(reflect.TypeOf(zero).Name()) + "_id"

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = placeholder(db.DriverName, i+1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		childTable, fk, strings.Join(placeholders, ", "))

	rows, err := db.SQL.Query(query, ids...)
	if err != nil {
		return fmt.Errorf("gai/orm: hasMany query failed: %w", err)
	}
	defer rows.Close()

	childItems, err := scanRowsDynamic(rows, childType)
	if err != nil {
		return err
	}

	// Group children by foreign key.
	grouped := make(map[any][]reflect.Value)
	for _, child := range childItems {
		fkVal := fieldValue(child, support.Camel(fk))
		grouped[fkVal] = append(grouped[fkVal], reflect.ValueOf(child.Interface()))
	}

	// Assign to parent models.
	for i := range models {
		v := reflect.ValueOf(&models[i]).Elem()
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
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

func loadBelongsTo[T any](db *DB, models []T, relation string, field reflect.StructField) error {
	relType := field.Type
	if relType.Kind() == reflect.Ptr {
		relType = relType.Elem()
	}

	fk := support.Snake(relation) + "_id"
	relTable := strings.ToLower(support.Plural(support.Snake(relType.Name())))

	ids := make([]any, 0, len(models))
	for _, m := range models {
		v := reflect.ValueOf(m)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
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

	query := fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)",
		relTable, strings.Join(placeholders, ", "))

	rows, err := db.SQL.Query(query, ids...)
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
		v := reflect.ValueOf(&models[i]).Elem()
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

func fieldValueReflect(v reflect.Value, name string) any {
	fv := v.FieldByName(name)
	if fv.IsValid() {
		return fv.Interface()
	}
	return nil
}

func parseRelationTag(tag string) (string, string) {
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, ":", 2)
		switch kv[0] {
		case "hasMany", "hasOne", "belongsTo":
			model := ""
			if len(kv) == 2 {
				model = kv[1]
			}
			return kv[0], model
		}
	}
	return "", ""
}
