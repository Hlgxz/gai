package generator

import (
	"strings"
	"text/template"

	"github.com/Hlgxz/gai/ai/schema"
)

const modelTemplate = `package models

import (
	"github.com/Hlgxz/gai/database/orm"
{{- range .Imports }}
	"{{ . }}"
{{- end }}
)

// {{ .Model }} represents the {{ .Table }} table.
type {{ .Model }} struct {
	orm.Model
{{- range .Fields }}
	{{ .GoName }}  {{ .GoType }}  ` + "`" + `json:"{{ .JSONName }}" gai:"column:{{ .Column }}{{ .TagExtra }}"` + "`" + `
{{- end }}
{{- range .Relations }}
	{{ .Name }}  {{ .GoType }}  ` + "`" + `json:"{{ .JSONName }},omitempty" gai:"{{ .RelType }}"` + "`" + `
{{- end }}
}

// TableName returns the database table name.
func ({{ .Receiver }} *{{ .Model }}) TableName() string {
	return "{{ .Table }}"
}
`

type modelData struct {
	Model     string
	Table     string
	Receiver  string
	Imports   []string
	Fields    []modelField
	Relations []modelRelation
}

type modelField struct {
	GoName  string
	GoType  string
	JSONName string
	Column  string
	TagExtra string
}

type modelRelation struct {
	Name    string
	GoType  string
	JSONName string
	RelType string
}

// GenerateModel produces the Go model file content from a schema.
func (g *Generator) GenerateModel(s *schema.Schema) (string, error) {
	imports := make(map[string]bool)
	var fields []modelField

	for _, f := range s.Fields {
		goName := toCamel(f.Name)
		goType := f.GoType()

		if pkg, ok := f.NeedsImport(); ok {
			imports[pkg] = true
		}

		var tagExtra string
		if f.Size > 0 {
			tagExtra += ";size:" + itoa(f.Size)
		}
		if f.Unique {
			tagExtra += ";unique"
		}
		if f.Nullable {
			tagExtra += ";nullable"
			if goType != "string" {
				goType = "*" + goType
			}
		}
		if f.Index {
			tagExtra += ";index"
		}

		fields = append(fields, modelField{
			GoName:   goName,
			GoType:   goType,
			JSONName: f.Name,
			Column:   f.Name,
			TagExtra: tagExtra,
		})
	}

	var relations []modelRelation
	for _, r := range s.Relations {
		rel := modelRelation{
			Name:    r.Model,
			JSONName: toSnake(r.Model),
			RelType: r.Type,
		}
		switch r.Type {
		case "hasMany":
			rel.Name = pluralize(r.Model)
			rel.JSONName = toSnake(pluralize(r.Model))
			rel.GoType = "[]" + r.Model
		case "hasOne", "belongsTo":
			rel.GoType = "*" + r.Model
		}
		relations = append(relations, rel)
	}

	importList := make([]string, 0, len(imports))
	for pkg := range imports {
		importList = append(importList, pkg)
	}

	data := modelData{
		Model:     s.Model,
		Table:     s.Table,
		Receiver:  strings.ToLower(s.Model[:1]),
		Imports:   importList,
		Fields:    fields,
		Relations: relations,
	}

	tmpl, err := template.New("model").Parse(modelTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
