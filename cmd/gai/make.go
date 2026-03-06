package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Hlgxz/gai/support"
)

func makeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Generate code scaffolding",
	}

	cmd.AddCommand(
		makeModelCmd(),
		makeControllerCmd(),
		makeMiddlewareCmd(),
		makeMigrationCmd(),
	)

	return cmd
}

func makeModelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "model [Name]",
		Short: "Generate a new model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := support.Camel(args[0])
			table := strings.ToLower(support.Plural(support.Snake(name)))
			return writeTemplate("app/models", support.Snake(name)+".go", modelStub(name, table))
		},
	}
}

func makeControllerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "controller [Name]",
		Short: "Generate a new controller",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := support.Camel(args[0])
			return writeTemplate("app/controllers", support.Snake(name)+"_controller.go", controllerStub(name))
		},
	}
}

func makeMiddlewareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "middleware [Name]",
		Short: "Generate a new middleware",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := support.Camel(args[0])
			return writeTemplate("app/middleware", support.Snake(name)+".go", middlewareStub(name))
		},
	}
}

func makeMigrationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migration [name]",
		Short: "Generate a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := support.Snake(args[0])
			ts := time.Now().Format("20060102150405")
			filename := ts + "_" + name + ".go"

			ensureMigrationRegistry("database/migrations")

			return writeTemplate("database/migrations", filename, migrationStub(ts+"_"+name))
		},
	}
}

func ensureMigrationRegistry(dir string) {
	path := filepath.Join(dir, "registry.go")
	if _, err := os.Stat(path); err == nil {
		return
	}
	os.MkdirAll(dir, 0o755)
	content := `package migrations

import "github.com/Hlgxz/gai/database/migration"

// Migrations collects all migrations registered via init() functions.
var Migrations []migration.Migration
`
	os.WriteFile(path, []byte(content), 0o644)
}

func writeTemplate(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Created: %s\n", path)
	return nil
}

func modelStub(name, table string) string {
	receiver := strings.ToLower(name[:1])
	return fmt.Sprintf(`package models

import (
	"github.com/Hlgxz/gai/database/orm"
)

// %s represents the %s table.
type %s struct {
	orm.Model
	// Add your fields here
}

// TableName returns the database table name.
func (%s *%s) TableName() string {
	return "%s"
}
`, name, table, name, receiver, name, table)
}

func controllerStub(name string) string {
	return fmt.Sprintf(`package controllers

import (
	"net/http"

	ghttp "github.com/Hlgxz/gai/http"
)

// %sController handles %s operations.
type %sController struct {
	// Add dependencies here
}

// New%sController creates a new controller instance.
func New%sController() *%sController {
	return &%sController{}
}

// Index lists resources.
func (ctrl *%sController) Index(c *ghttp.Context) {
	c.Success(nil)
}

// Show returns a single resource.
func (ctrl *%sController) Show(c *ghttp.Context) {
	id := c.Param("id")
	c.Success(map[string]string{"id": id})
}

// Store creates a new resource.
func (ctrl *%sController) Store(c *ghttp.Context) {
	c.JSON(http.StatusCreated, map[string]any{
		"code": 0, "message": "created",
	})
}

// Update modifies an existing resource.
func (ctrl *%sController) Update(c *ghttp.Context) {
	c.Success(nil)
}

// Destroy deletes a resource.
func (ctrl *%sController) Destroy(c *ghttp.Context) {
	c.NoContent()
}
`, name, name, name, name, name, name, name, name, name, name, name, name)
}

func middlewareStub(name string) string {
	return fmt.Sprintf(`package middleware

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// %s returns a new %s middleware handler.
func %s() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		// Add your middleware logic here
		c.Next()
	}
}
`, name, name, name)
}

func migrationStub(name string) string {
	return fmt.Sprintf(`package migrations

import (
	"github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/migration"
)

func init() {
	Migrations = append(Migrations, migration.Migration{
		Name: "%s",
		Up: func(drv driver.Driver) string {
			// Write your UP migration SQL here
			return ""
		},
		Down: func(drv driver.Driver) string {
			// Write your DOWN migration SQL here
			return ""
		},
	})
}
`, name)
}
