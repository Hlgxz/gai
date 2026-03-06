package schema

// Schema is the root structure parsed from a YAML schema file.
// It describes a complete business entity: model, fields, API, and relations.
type Schema struct {
	Model     string     `yaml:"model"`
	Table     string     `yaml:"table"`
	Fields    []Field    `yaml:"fields"`
	API       APIConfig  `yaml:"api"`
	Relations []Relation `yaml:"relations"`
}

// Field describes a single column/attribute of the model.
type Field struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Size     int      `yaml:"size,omitempty"`
	Unique   bool     `yaml:"unique,omitempty"`
	Nullable bool     `yaml:"nullable,omitempty"`
	Index    bool     `yaml:"index,omitempty"`
	Default  string   `yaml:"default,omitempty"`
	Rules    string   `yaml:"rules,omitempty"`
	Values   []string `yaml:"values,omitempty"` // for enum types
}

// APIConfig describes the REST API endpoints to generate.
type APIConfig struct {
	Prefix  string   `yaml:"prefix"`
	Actions []string `yaml:"actions"` // index, show, store, update, destroy
	Auth    string   `yaml:"auth"`    // guard name
}

// Relation describes a model relationship.
type Relation struct {
	Type  string `yaml:"type"`  // hasMany, hasOne, belongsTo
	Model string `yaml:"model"` // related model name
	FK    string `yaml:"fk,omitempty"`
}

// GoType maps a Gai schema type to a Go type string.
func (f *Field) GoType() string {
	switch f.Type {
	case "string", "text", "enum":
		return "string"
	case "int", "integer":
		return "int"
	case "bigint":
		return "int64"
	case "float":
		return "float32"
	case "double", "decimal":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "date", "datetime", "timestamp":
		return "time.Time"
	case "json":
		return "json.RawMessage"
	default:
		return "string"
	}
}

// NeedsImport returns true if the field type requires an additional import.
func (f *Field) NeedsImport() (string, bool) {
	switch f.Type {
	case "date", "datetime", "timestamp":
		return "time", true
	case "json":
		return "encoding/json", true
	default:
		return "", false
	}
}
