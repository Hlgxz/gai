package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hlgxz/gai/ai/schema"
)

// Generator orchestrates code generation from schema definitions.
type Generator struct {
	OutputDir string
	Module    string // Go module path, e.g. "github.com/user/myapp"
}

// NewGenerator creates a generator writing to the specified output directory.
func NewGenerator(outputDir, module string) *Generator {
	return &Generator{OutputDir: outputDir, Module: module}
}

// GenerateAll generates model, controller, migration, and routes from a schema.
func (g *Generator) GenerateAll(s *schema.Schema) error {
	files := []struct {
		name string
		fn   func(*schema.Schema) (string, error)
		path string
	}{
		{"model", g.GenerateModel, filepath.Join("app", "models")},
		{"controller", g.GenerateController, filepath.Join("app", "controllers")},
		{"migration", g.GenerateMigration, filepath.Join("database", "migrations")},
		{"routes", g.GenerateRoutes, filepath.Join("routes")},
	}

	for _, f := range files {
		content, err := f.fn(s)
		if err != nil {
			return fmt.Errorf("gai/generator: %s generation failed: %w", f.name, err)
		}

		dir := filepath.Join(g.OutputDir, f.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("gai/generator: mkdir %s failed: %w", dir, err)
		}

		if f.name == "migration" {
			ensureMigrationRegistry(dir)
		}

		var filename string
		switch f.name {
		case "model":
			filename = toSnake(s.Model) + ".go"
		case "controller":
			filename = toSnake(s.Model) + "_controller.go"
		case "migration":
			filename = migrationTimestamp() + "_create_" + s.Table + "_table.go"
		case "routes":
			filename = toSnake(s.Model) + "_routes.go"
		}

		outPath := filepath.Join(dir, filename)
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("gai/generator: write %s failed: %w", outPath, err)
		}

		fmt.Printf("  created: %s\n", outPath)
	}

	return nil
}

func ensureMigrationRegistry(dir string) {
	path := filepath.Join(dir, "registry.go")
	if _, err := os.Stat(path); err == nil {
		return
	}
	content := `package migrations

import "github.com/Hlgxz/gai/database/migration"

// Migrations collects all migrations registered via init() functions.
var Migrations []migration.Migration
`
	os.WriteFile(path, []byte(content), 0o644)
}

// GenerateFromDir processes all schema files in a directory.
func (g *Generator) GenerateFromDir(schemaDir string) error {
	schemas, err := schema.ParseDir(schemaDir)
	if err != nil {
		return err
	}

	for _, s := range schemas {
		fmt.Printf("Generating code for model: %s\n", s.Model)
		if err := g.GenerateAll(s); err != nil {
			return err
		}
	}

	return nil
}
