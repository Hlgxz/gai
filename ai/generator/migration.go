package generator

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Hlgxz/gai/ai/schema"
	"github.com/Hlgxz/gai/support"
)

const migrationTemplate = `package migrations

import (
	"github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/migration"
)

func init() {
	Register(migration.Migration{
		Name: "{{ .Name }}",
		Up: func(drv driver.Driver) string {
			b := migration.NewBlueprint("{{ .Table }}", drv)
			b.ID()
			{{- range .Fields }}
			b.{{ .BlueprintCall }}
			{{- end }}
			b.Timestamps()
			b.SoftDeletes()
			return b.ToCreateSQL()
		},
		Down: func(drv driver.Driver) string {
			b := migration.NewBlueprint("{{ .Table }}", drv)
			return b.ToDropSQL()
		},
	})
}
`

type migrationData struct {
	Name   string
	Table  string
	Fields []migrationField
}

type migrationField struct {
	BlueprintCall string
}

// GenerateMigration produces the Go migration file content.
func (g *Generator) GenerateMigration(s *schema.Schema) (string, error) {
	var fields []migrationField

	for _, f := range s.Fields {
		call := fieldToBlueprintCall(f)
		fields = append(fields, migrationField{BlueprintCall: call})
	}

	data := migrationData{
		Name:   migrationTimestamp() + "_create_" + s.Table + "_table",
		Table:  s.Table,
		Fields: fields,
	}

	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fieldToBlueprintCall(f schema.Field) string {
	var call string
	name := fmt.Sprintf("%q", f.Name)

	switch f.Type {
	case "string":
		size := f.Size
		if size <= 0 {
			size = 255
		}
		call = fmt.Sprintf("String(%s, %d)", name, size)
	case "text":
		call = fmt.Sprintf("Text(%s)", name)
	case "int", "integer":
		call = fmt.Sprintf("Integer(%s)", name)
	case "bigint":
		call = fmt.Sprintf("BigInteger(%s)", name)
	case "float":
		call = fmt.Sprintf("Float(%s)", name)
	case "decimal":
		call = fmt.Sprintf("Decimal(%s)", name)
	case "bool", "boolean":
		call = fmt.Sprintf("Boolean(%s)", name)
	case "datetime", "timestamp", "date":
		call = fmt.Sprintf("DateTime(%s)", name)
	case "json":
		call = fmt.Sprintf("JSON(%s)", name)
	case "enum":
		call = fmt.Sprintf("String(%s, 50)", name)
	default:
		call = fmt.Sprintf("String(%s, 255)", name)
	}

	if f.Nullable {
		call += ".SetNullable()"
	}
	if f.Unique {
		call += ".SetUnique()"
	}
	if f.Index {
		call += ".SetIndex()"
	}
	if f.Default != "" {
		call += fmt.Sprintf(".SetDefault(%q)", f.Default)
	}

	return call
}

func migrationTimestamp() string {
	return time.Now().Format("20060102150405")
}

// ---------------------------------------------------------- Helpers

func toSnake(s string) string { return support.Snake(s) }
func toCamel(s string) string { return support.Camel(s) }
func pluralize(s string) string { return support.Plural(s) }

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
