package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Hlgxz/gai/ai/generator"
	"github.com/Hlgxz/gai/ai/schema"
)

func generateCmd() *cobra.Command {
	var schemaPath string
	var module string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from schema definitions",
		Long:  "Generate model, controller, migration, and routes from YAML schema files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if module == "" {
				module = detectModule()
			}
			if outputDir == "" {
				outputDir = "."
			}

			gen := generator.NewGenerator(outputDir, module)

			info, err := os.Stat(schemaPath)
			if err != nil {
				return fmt.Errorf("schema path not found: %s", schemaPath)
			}

			if info.IsDir() {
				fmt.Printf("Generating from all schemas in: %s\n", schemaPath)
				return gen.GenerateFromDir(schemaPath)
			}

			fmt.Printf("Generating from schema: %s\n", schemaPath)
			s, err := schema.ParseFile(schemaPath)
			if err != nil {
				return err
			}
			return gen.GenerateAll(s)
		},
	}

	cmd.Flags().StringVarP(&schemaPath, "schema", "s", "schemas", "Path to schema file or directory")
	cmd.Flags().StringVarP(&module, "module", "m", "", "Go module path (auto-detected from go.mod)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory")

	return cmd
}

// detectModule reads the go.mod file to find the module path.
func detectModule() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "myapp"
	}
	for _, line := range splitLines(string(data)) {
		if len(line) > 7 && line[:7] == "module " {
			return line[7:]
		}
	}
	return "myapp"
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// Ensure the schemas directory is checked relative to cwd.
func init() {
	if _, err := os.Stat("schemas"); os.IsNotExist(err) {
		p, _ := filepath.Abs("schemas")
		_ = p
	}
}
