package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hlgxz/gai/ai/generator"
	"github.com/Hlgxz/gai/ai/schema"
)

func TestGenerateControllerWithDB(t *testing.T) {
	s := &schema.Schema{
		Model: "User",
		Table: "users",
		Fields: []schema.Field{
			{Name: "name", Type: "string", Rules: "required|min:2"},
			{Name: "email", Type: "string", Unique: true, Rules: "required|email"},
		},
		API: schema.APIConfig{Prefix: "/api/users", Actions: []string{"store"}},
	}
	g := generator.NewGenerator(t.TempDir(), "example.com/demo")
	src, err := g.GenerateController(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, `validator.WithDB(ctrl.DB)`) {
		t.Fatalf("expected WithDB:\n%s", src)
	}
	if !strings.Contains(src, `unique:users,email`) {
		t.Fatalf("expected unique rule:\n%s", src)
	}
}

func TestRegisterGeneratedRoutes(t *testing.T) {
	dir := t.TempDir()
	routesDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routesDir, "routes.go"), []byte(`package routes

import "github.com/Hlgxz/gai"

func Register(app *gai.Application) {
	r := app.Router()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &schema.Schema{
		Model: "Post",
		Table: "posts",
		Fields: []schema.Field{
			{Name: "title", Type: "string"},
		},
		API: schema.APIConfig{Prefix: "/posts", Auth: "jwt"},
	}
	g := generator.NewGenerator(dir, "example.com/demo")
	if err := g.GenerateAll(s); err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(routesDir, "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(generated)
	if !strings.Contains(body, "RegisterPostRoutes") {
		t.Fatalf("generated.go missing RegisterPostRoutes:\n%s", body)
	}
	if !strings.Contains(body, "gai-route:Post auth") {
		t.Fatalf("missing route marker:\n%s", body)
	}

	routes, err := os.ReadFile(filepath.Join(routesDir, "routes.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routes), "registerGenerated(") {
		t.Fatalf("routes.go not wired:\n%s", routes)
	}
}
