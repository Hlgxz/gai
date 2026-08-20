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
	Migrations = append(Migrations, migration.Migration{
		Name: "{{ .Name }}",
		Up: func(drv driver.Driver) string {
			b := migration.NewBlueprint("{{ .Table }}", drv)
			b.ID()
			{{- range .Fields }}
			b.{{ .BlueprintCall }}
			{{- end }}
			{{- range .Relations }}
			{{ . }}
			{{- end }}
			b.Timestamps()
			b.SoftDeletes()
			sql := b.ToCreateSQL()
			{{- if .PivotSQL }}
			{{ .PivotSQL }}
			{{- end }}
			return sql
		},
		Down: func(drv driver.Driver) string {
			b := migration.NewBlueprint("{{ .Table }}", drv)
			sql := b.ToDropSQL()
			{{- if .PivotTable }}
			sql += "; DROP TABLE IF EXISTS {{ .PivotTable }}"
			{{- end }}
			return sql
		},
	})
}
`

type migrationData struct {
	Name       string
	Table      string
	Fields     []migrationField
	Relations  []string
	PivotSQL   string
	PivotTable string
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
	data.Relations, data.PivotSQL, data.PivotTable = relationBlueprint(s)

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

func relationBlueprint(s *schema.Schema) (rels []string, pivotSQL, pivotTable string) {
	for _, r := range s.Relations {
		switch strings.ToLower(r.Type) {
		case "belongsto":
			fk := r.FK
			if fk == "" {
				fk = support.Snake(r.Model) + "_id"
			}
			table := support.Plural(support.Snake(r.Model))
			rels = append(rels,
				fmt.Sprintf("b.BigInteger(%q)", fk),
				fmt.Sprintf(`b.Foreign(%q).References("id").On(%q)`, fk, table),
			)
		case "belongstomany":
			a, b := support.Snake(s.Model), support.Snake(r.Model)
			if a > b {
				a, b = b, a
			}
			pivot := r.Pivot
			if pivot == "" {
				pivot = a + "_" + b
			}
			fk := r.FK
			if fk == "" {
				fk = support.Snake(s.Model) + "_id"
			}
			relatedFK := r.RelatedFK
			if relatedFK == "" {
				relatedFK = support.Snake(r.Model) + "_id"
			}
			pivotTable = pivot
			pivotSQL = fmt.Sprintf("pivot := migration.NewBlueprint(%q, drv)\n\t\t\tpivot.BigInteger(%q)\n\t\t\tpivot.BigInteger(%q)\n\t\t\tsql += \";\\n\" + pivot.ToCreateSQL()",
				pivot, fk, relatedFK)
		}
	}
	return
}

func migrationTimestamp() string {
	return time.Now().Format("20060102150405")
}

// ---------------------------------------------------------- Helpers

func toSnake(s string) string   { return support.Snake(s) }
func toCamel(s string) string   { return support.Camel(s) }
func pluralize(s string) string { return support.Plural(s) }

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
