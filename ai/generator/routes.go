package generator

import (
	"strings"
	"text/template"

	"github.com/Hlgxz/gai/ai/schema"
)

const routesTemplate = `package routes

import (
	"github.com/Hlgxz/gai/database/orm"
	"github.com/Hlgxz/gai/router"
	{{- if .Auth }}
	"github.com/Hlgxz/gai/auth"
	{{- end }}
	"{{ .Module }}/app/controllers"
)

// Register{{ .Model }}Routes sets up the {{ .Model }} resource routes.
func Register{{ .Model }}Routes(r *router.Router, db *orm.DB{{ if .Auth }}, authMgr *auth.Manager{{ end }}) {
	ctrl := controllers.New{{ .Model }}Controller(db)

	r.Group("{{ .Prefix }}", func(g *router.Group) {
		{{- if .Auth }}
		g.Use(authMgr.Middleware("{{ .Auth }}"))
		{{- end }}
		{{- if .HasAction "index" }}
		g.Get("", ctrl.Index)
		{{- end }}
		{{- if .HasAction "store" }}
		g.Post("", ctrl.Store)
		{{- end }}
		{{- if .HasAction "show" }}
		g.Get("/:id", ctrl.Show)
		{{- end }}
		{{- if .HasAction "update" }}
		g.Put("/:id", ctrl.Update)
		{{- end }}
		{{- if .HasAction "destroy" }}
		g.Delete("/:id", ctrl.Destroy)
		{{- end }}
	})
}
`

type routesData struct {
	Module  string
	Model   string
	Prefix  string
	Auth    string
	Actions []string
}

func (d routesData) HasAction(name string) bool {
	for _, a := range d.Actions {
		if a == name {
			return true
		}
	}
	return false
}

// GenerateRoutes produces the Go route registration file content.
func (g *Generator) GenerateRoutes(s *schema.Schema) (string, error) {
	prefix := s.API.Prefix
	if prefix == "" {
		prefix = "/api/" + strings.ToLower(pluralize(s.Model))
	}

	actions := s.API.Actions
	if len(actions) == 0 {
		actions = []string{"index", "show", "store", "update", "destroy"}
	}

	data := routesData{
		Module:  g.Module,
		Model:   s.Model,
		Prefix:  prefix,
		Auth:    s.API.Auth,
		Actions: actions,
	}

	funcMap := template.FuncMap{
		"hasAction": data.HasAction,
	}

	tmpl, err := template.New("routes").Funcs(funcMap).Parse(routesTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
