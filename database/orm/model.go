package orm

import (
	"reflect"
	"strings"
	"time"

	"github.com/Hlgxz/gai/support"
)

// Model is embedded into user-defined models to provide common fields
// (ID, timestamps) and metadata, similar to Laravel's Eloquent base model.
type Model struct {
	ID        uint64     `json:"id" gai:"column:id;primaryKey"`
	CreatedAt time.Time  `json:"created_at" gai:"column:created_at"`
	UpdatedAt time.Time  `json:"updated_at" gai:"column:updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gai:"column:deleted_at;softDelete"`
}

// FieldInfo holds parsed metadata from struct tags for a single field.
type FieldInfo struct {
	Name       string
	Column     string
	GoType     reflect.Type
	GaiType    string
	Size       int
	PrimaryKey bool
	Unique     bool
	Nullable   bool
	Index      bool
	Default    string
	SoftDelete bool
	Relation   string // hasMany, belongsTo, hasOne
	RelModel   string
}

// TableName derives a table name from a struct type following convention:
// "User" -> "users", "BlogPost" -> "blog_posts".
func TableName(model any) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.ToLower(support.Plural(support.Snake(t.Name())))
}

// ParseFields extracts FieldInfo from a struct's `gai` tags.
func ParseFields(model any) []FieldInfo {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var fields []FieldInfo

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Recurse into embedded structs (like gai.Model).
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			embedded := reflect.New(f.Type).Elem().Interface()
			fields = append(fields, ParseFields(embedded)...)
			continue
		}

		tag := f.Tag.Get("gai")
		if tag == "-" {
			continue
		}

		info := FieldInfo{
			Name:   f.Name,
			Column: support.Snake(f.Name),
			GoType: f.Type,
		}

		if tag != "" {
			parseTag(tag, &info)
		}

		// Skip slice/struct fields that look like relations.
		if info.Relation != "" {
			fields = append(fields, info)
			continue
		}
		if f.Type.Kind() == reflect.Slice || (f.Type.Kind() == reflect.Struct &&
			f.Type != reflect.TypeOf(time.Time{}) && f.Type != reflect.TypeOf(Model{})) {
			continue
		}

		fields = append(fields, info)
	}

	return fields
}

// Columns returns just the column names for non-relation fields.
func Columns(model any) []string {
	fields := ParseFields(model)
	cols := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Relation == "" {
			cols = append(cols, f.Column)
		}
	}
	return cols
}

func parseTag(tag string, info *FieldInfo) {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, ":", 2)
		key := kv[0]
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}

		switch key {
		case "column":
			info.Column = val
		case "primaryKey":
			info.PrimaryKey = true
		case "unique":
			info.Unique = true
		case "nullable":
			info.Nullable = true
		case "index":
			info.Index = true
		case "size":
			if n := parseInt(val); n > 0 {
				info.Size = n
			}
		case "type":
			info.GaiType = val
		case "default":
			info.Default = val
		case "softDelete":
			info.SoftDelete = true
			info.Nullable = true
		case "hasMany", "hasOne", "belongsTo":
			info.Relation = key
			info.RelModel = val
		}
	}
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
