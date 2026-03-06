package generator

import (
	"strings"
	"text/template"

	"github.com/Hlgxz/gai/ai/schema"
)

const controllerTemplate = `package controllers

import (
	"fmt"
	"net/http"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/database/orm"
	"{{ .Module }}/app/models"
)

// {{ .Model }}Controller handles CRUD operations for {{ .Model }}.
type {{ .Model }}Controller struct {
	DB *orm.DB
}

// New{{ .Model }}Controller creates a new controller instance.
func New{{ .Model }}Controller(db *orm.DB) *{{ .Model }}Controller {
	return &{{ .Model }}Controller{DB: db}
}
{{- if .HasAction "index" }}

// Index lists all {{ .PluralLower }} with pagination.
func (ctrl *{{ .Model }}Controller) Index(c *ghttp.Context) {
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 20)

	result, err := orm.Paginate[models.{{ .Model }}](
		orm.Query[models.{{ .Model }}](ctrl.DB).
			OrderBy("created_at", "DESC"),
		page, perPage,
	)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(result)
}
{{- end }}
{{- if .HasAction "show" }}

// Show returns a single {{ .ModelLower }} by ID.
func (ctrl *{{ .Model }}Controller) Show(c *ghttp.Context) {
	id := c.ParamInt("id")
	item, err := orm.First[models.{{ .Model }}](
		orm.Query[models.{{ .Model }}](ctrl.DB).Where("id", "=", id),
	)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		c.Error(http.StatusNotFound, "{{ .Model }} not found")
		return
	}
	c.Success(item)
}
{{- end }}
{{- if .HasAction "store" }}

// Store creates a new {{ .ModelLower }}.
func (ctrl *{{ .Model }}Controller) Store(c *ghttp.Context) {
	var input map[string]any
	if err := c.BindJSON(&input); err != nil {
		c.Error(http.StatusBadRequest, "Invalid JSON")
		return
	}

	{{- if .ValidationRules }}
	validator := ghttp.NewValidator(input, map[string]string{
		{{- range .ValidationRules }}
		"{{ .Field }}": "{{ .Rules }}",
		{{- end }}
	})
	if errs := validator.Validate(); errs != nil {
		c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"code":    422,
			"message": "Validation failed",
			"errors":  errs,
		})
		return
	}
	{{- end }}

	item := &models.{{ .Model }}{}
	{{- range .Fields }}
	if v, ok := input["{{ .Name }}"]; ok {
		if typed, ok := v.({{ .GoType }}); ok {
			item.{{ .GoName }} = typed
		} else {
			c.Error(http.StatusBadRequest, fmt.Sprintf("invalid type for field {{ .Name }}: expected {{ .GoType }}, got %T", v))
			return
		}
	}
	{{- end }}

	result, err := orm.Create[models.{{ .Model }}](ctrl.DB, item)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, map[string]any{
		"code":    0,
		"message": "ok",
		"data":    result,
	})
}
{{- end }}
{{- if .HasAction "update" }}

// Update modifies an existing {{ .ModelLower }}.
func (ctrl *{{ .Model }}Controller) Update(c *ghttp.Context) {
	id := c.ParamInt("id")
	item, err := orm.First[models.{{ .Model }}](
		orm.Query[models.{{ .Model }}](ctrl.DB).Where("id", "=", id),
	)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		c.Error(http.StatusNotFound, "{{ .Model }} not found")
		return
	}

	var input map[string]any
	if err := c.BindJSON(&input); err != nil {
		c.Error(http.StatusBadRequest, "Invalid JSON")
		return
	}

	{{- range .Fields }}
	if v, ok := input["{{ .Name }}"]; ok {
		if typed, ok := v.({{ .GoType }}); ok {
			item.{{ .GoName }} = typed
		} else {
			c.Error(http.StatusBadRequest, fmt.Sprintf("invalid type for field {{ .Name }}: expected {{ .GoType }}, got %T", v))
			return
		}
	}
	{{- end }}

	if err := orm.Update[models.{{ .Model }}](ctrl.DB, item); err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(item)
}
{{- end }}
{{- if .HasAction "destroy" }}

// Destroy deletes a {{ .ModelLower }} by ID.
func (ctrl *{{ .Model }}Controller) Destroy(c *ghttp.Context) {
	id := c.ParamInt("id")
	item, err := orm.First[models.{{ .Model }}](
		orm.Query[models.{{ .Model }}](ctrl.DB).Where("id", "=", id),
	)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		c.Error(http.StatusNotFound, "{{ .Model }} not found")
		return
	}

	if err := orm.Delete[models.{{ .Model }}](ctrl.DB, item); err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.NoContent()
}
{{- end }}

// Ensure {{ .Model }}Controller satisfies the ResourceController interface.
var _ interface {
	Index(c *ghttp.Context)
	Show(c *ghttp.Context)
	Store(c *ghttp.Context)
	Update(c *ghttp.Context)
	Destroy(c *ghttp.Context)
} = (*{{ .Model }}Controller)(nil)
`

type controllerData struct {
	Module          string
	Model           string
	ModelLower      string
	PluralLower     string
	Actions         []string
	Fields          []controllerField
	ValidationRules []validationRule
}

type controllerField struct {
	Name   string
	GoName string
	GoType string
}

type validationRule struct {
	Field string
	Rules string
}

func (d controllerData) HasAction(name string) bool {
	for _, a := range d.Actions {
		if a == name {
			return true
		}
	}
	return false
}

// GenerateController produces the Go controller file content.
func (g *Generator) GenerateController(s *schema.Schema) (string, error) {
	var fields []controllerField
	var rules []validationRule

	for _, f := range s.Fields {
		fields = append(fields, controllerField{
			Name:   f.Name,
			GoName: toCamel(f.Name),
			GoType: f.GoType(),
		})
		if f.Rules != "" {
			rules = append(rules, validationRule{
				Field: f.Name,
				Rules: f.Rules,
			})
		}
	}

	actions := s.API.Actions
	if len(actions) == 0 {
		actions = []string{"index", "show", "store", "update", "destroy"}
	}

	data := controllerData{
		Module:          g.Module,
		Model:           s.Model,
		ModelLower:      strings.ToLower(s.Model),
		PluralLower:     strings.ToLower(pluralize(s.Model)),
		Actions:         actions,
		Fields:          fields,
		ValidationRules: rules,
	}

	funcMap := template.FuncMap{
		"hasAction": data.HasAction,
	}

	tmpl, err := template.New("controller").Funcs(funcMap).Parse(controllerTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
