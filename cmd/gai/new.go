package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCmd() *cobra.Command {
	var module string
	var port int

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Gai project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if module == "" {
				module = "github.com/yourname/" + name
			}
			return createProject(name, module, port)
		},
	}

	cmd.Flags().StringVarP(&module, "module", "m", "", "Go module path (default: github.com/yourname/<name>)")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP server port")

	return cmd
}

func createProject(name, module string, port int) error {
	fmt.Printf("Creating new Gai project: %s\n", name)

	dirs := []string{
		"app/controllers",
		"app/models",
		"config",
		"database/migrations",
		"routes",
		"schemas",
		"storage/logs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(name, dir), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"go.mod":         goModContent(module),
		"main.go":        mainGoContent(module, port),
		".env":           envContent(port),
		"config/app.yaml": appYamlContent(name, port),
		"schemas/.gitkeep": "",
		"database/migrations/.gitkeep": "",
		"routes/routes.go": routesGoContent(module),
		".gitignore":      gitignoreContent(),
	}

	for path, content := range files {
		fullPath := filepath.Join(name, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Printf("  created: %s\n", path)
	}

	fmt.Printf("\nProject %s created successfully!\n\n", name)
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  go mod tidy")
	fmt.Println("  gai serve")
	fmt.Println()

	return nil
}

func goModContent(module string) string {
	return fmt.Sprintf(`module %s

go 1.22

require github.com/Hlgxz/gai v0.1.0
`, module)
}

func mainGoContent(module string, port int) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"

	"github.com/Hlgxz/gai"
	"%s/routes"
)

func main() {
	app := gai.New()
	app.LoadConfig("config")
	app.UseDefaults()

	routes.Register(app)

	addr := fmt.Sprintf(":%%d", %d)
	log.Fatal(app.Serve(addr))
}
`, module, port)
}

func envContent(port int) string {
	return fmt.Sprintf(`APP_ENV=development
APP_PORT=%d
APP_DEBUG=true

DB_DRIVER=sqlite
DB_DATABASE=storage/database.db

JWT_SECRET=change-me-to-a-random-string
JWT_TTL=7200

# WECHAT_APP_ID=
# WECHAT_APP_SECRET=
`, port)
}

func appYamlContent(name string, port int) string {
	return fmt.Sprintf(`name: %s
port: %d
env: ${APP_ENV:development}
debug: ${APP_DEBUG:true}

database:
  driver: ${DB_DRIVER:sqlite}
  dsn: ${DB_DATABASE:storage/database.db}

auth:
  default: jwt
  guards:
    jwt:
      driver: jwt
      secret: ${JWT_SECRET}
      ttl: ${JWT_TTL:7200}
`, name, port)
}

func routesGoContent(module string) string {
	_ = module
	return `package routes

import (
	"github.com/Hlgxz/gai"
	ghttp "github.com/Hlgxz/gai/http"
)

// Register sets up all application routes.
func Register(app *gai.Application) {
	r := app.Router()

	r.Get("/", func(c *ghttp.Context) {
		c.Success(map[string]string{
			"framework": "Gai",
			"version":   "0.1.0",
			"message":   "Welcome to Gai! Define once, generate everything.",
		})
	})

	r.Get("/health", func(c *ghttp.Context) {
		c.OK(map[string]string{"status": "ok"})
	})
}
`
}

func gitignoreContent() string {
	return strings.TrimSpace(`
# Binaries
*.exe
*.dll
*.so
*.dylib

# IDE
.idea/
.vscode/
*.swp
*.swo

# Environment
.env
.env.local

# Storage
storage/logs/*
storage/*.db
!storage/.gitkeep

# Build
/tmp
/dist
`) + "\n"
}
