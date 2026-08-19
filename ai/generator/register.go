package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hlgxz/gai/ai/schema"
)

type generatedRoute struct {
	Model string
	Auth  bool
}

func (g *Generator) registerRoutes(s *schema.Schema) error {
	routesDir := filepath.Join(g.OutputDir, "routes")
	if err := os.MkdirAll(routesDir, 0o755); err != nil {
		return err
	}

	generatedPath := filepath.Join(routesDir, "generated.go")
	existing := []generatedRoute{}
	if data, err := os.ReadFile(generatedPath); err == nil {
		existing = parseGeneratedRoutes(string(data))
	}

	found := false
	auth := s.API.Auth != ""
	for i, r := range existing {
		if r.Model == s.Model {
			existing[i].Auth = auth
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, generatedRoute{Model: s.Model, Auth: auth})
	}

	src := renderGeneratedRoutes(existing)
	if err := os.WriteFile(generatedPath, []byte(src), 0o644); err != nil {
		return err
	}

	return ensureRegisterCall(filepath.Join(routesDir, "routes.go"))
}

func parseGeneratedRoutes(src string) []generatedRoute {
	var out []generatedRoute
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "// gai-route:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "// gai-route:"))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		r := generatedRoute{Model: parts[0]}
		if len(parts) > 1 && parts[1] == "auth" {
			r.Auth = true
		}
		out = append(out, r)
	}
	return out
}

func renderGeneratedRoutes(routes []generatedRoute) string {
	var b strings.Builder
	b.WriteString("package routes\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"github.com/Hlgxz/gai\"\n")
	b.WriteString("\t\"github.com/Hlgxz/gai/auth\"\n")
	b.WriteString("\t\"github.com/Hlgxz/gai/database/orm\"\n")
	b.WriteString("\t\"github.com/Hlgxz/gai/router\"\n")
	b.WriteString(")\n\n")
	for _, r := range routes {
		if r.Auth {
			fmt.Fprintf(&b, "// gai-route:%s auth\n", r.Model)
		} else {
			fmt.Fprintf(&b, "// gai-route:%s\n", r.Model)
		}
	}
	b.WriteString("\nfunc registerGenerated(app *gai.Application, r *router.Router) {\n")
	if len(routes) == 0 {
		b.WriteString("}\n")
		return formatGenerated(b.String())
	}
	b.WriteString("\tvar db *orm.DB\n")
	b.WriteString("\tif app.Has(\"db\") {\n")
	b.WriteString("\t\tdb = gai.Make[*orm.DB](app.Container, \"db\")\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvar authMgr *auth.Manager\n")
	b.WriteString("\tif app.Has(\"auth\") {\n")
	b.WriteString("\t\tauthMgr = gai.Make[*auth.Manager](app.Container, \"auth\")\n")
	b.WriteString("\t}\n")
	for _, rt := range routes {
		if rt.Auth {
			fmt.Fprintf(&b, "\tif db != nil && authMgr != nil {\n\t\tRegister%sRoutes(r, db, authMgr)\n\t}\n", rt.Model)
		} else {
			fmt.Fprintf(&b, "\tif db != nil {\n\t\tRegister%sRoutes(r, db)\n\t}\n", rt.Model)
		}
	}
	b.WriteString("}\n")
	return formatGenerated(b.String())
}

func formatGenerated(src string) string {
	out, err := format.Source([]byte(src))
	if err != nil {
		return src
	}
	return string(out)
}

func ensureRegisterCall(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	src := string(data)
	if strings.Contains(src, "registerGenerated(") {
		return nil
	}
	old := "r := app.Router()"
	if !strings.Contains(src, old) {
		return fmt.Errorf("gai/generator: cannot wire registerGenerated into %s (no app.Router() call)", path)
	}
	src = strings.Replace(src, old, old+"\n\tregisterGenerated(app, r)", 1)
	formatted := formatGenerated(src)
	return os.WriteFile(path, []byte(formatted), 0o644)
}
