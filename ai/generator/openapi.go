package generator

import (
	"fmt"
	"strings"

	"github.com/Hlgxz/gai/ai/schema"
	"github.com/Hlgxz/gai/support"
)

// GenerateOpenAPI builds an OpenAPI 3 document from schemas.
func GenerateOpenAPI(title string, schemas []*schema.Schema) string {
	if title == "" {
		title = "Gai API"
	}
	var b strings.Builder
	b.WriteString("openapi: 3.0.3\n")
	b.WriteString("info:\n")
	b.WriteString(fmt.Sprintf("  title: %s\n", title))
	b.WriteString("  version: 1.0.0\n")
	b.WriteString("paths:\n")

	for _, s := range schemas {
		prefix := s.API.Prefix
		if prefix == "" {
			prefix = "/api/v1/" + s.Table
		}
		actions := s.API.Actions
		if len(actions) == 0 {
			actions = []string{"index", "show", "store", "update", "destroy"}
		}
		set := make(map[string]bool, len(actions))
		for _, a := range actions {
			set[a] = true
		}
		schemaName := s.Model
		if set["index"] || set["store"] {
			b.WriteString(fmt.Sprintf("  %s:\n", prefix))
			if set["index"] {
				writeOp(&b, "get", "List "+support.Plural(s.Model), schemaName, false, s.API.Auth)
			}
			if set["store"] {
				writeOp(&b, "post", "Create "+s.Model, schemaName, true, s.API.Auth)
			}
		}
		if set["show"] || set["update"] || set["destroy"] {
			b.WriteString(fmt.Sprintf("  %s/{id}:\n", prefix))
			b.WriteString("    parameters:\n")
			b.WriteString("      - name: id\n")
			b.WriteString("        in: path\n")
			b.WriteString("        required: true\n")
			b.WriteString("        schema: { type: integer }\n")
			if set["show"] {
				writeOp(&b, "get", "Get "+s.Model, schemaName, false, s.API.Auth)
			}
			if set["update"] {
				writeOp(&b, "put", "Update "+s.Model, schemaName, true, s.API.Auth)
			}
			if set["destroy"] {
				writeOp(&b, "delete", "Delete "+s.Model, schemaName, false, s.API.Auth)
			}
		}
	}

	b.WriteString("components:\n")
	b.WriteString("  schemas:\n")
	for _, s := range schemas {
		b.WriteString(fmt.Sprintf("    %s:\n", s.Model))
		b.WriteString("      type: object\n")
		b.WriteString("      properties:\n")
		b.WriteString("        id: { type: integer }\n")
		for _, f := range s.Fields {
			b.WriteString(fmt.Sprintf("        %s: { type: %s }\n", f.Name, openAPIType(f.Type)))
		}
	}
	b.WriteString("  securitySchemes:\n")
	b.WriteString("    bearerAuth:\n")
	b.WriteString("      type: http\n")
	b.WriteString("      scheme: bearer\n")
	b.WriteString("      bearerFormat: JWT\n")
	return b.String()
}

func writeOp(b *strings.Builder, method, summary, schema string, hasBody bool, auth string) {
	b.WriteString(fmt.Sprintf("    %s:\n", method))
	b.WriteString(fmt.Sprintf("      summary: %s\n", summary))
	if auth != "" {
		b.WriteString("      security: [{ bearerAuth: [] }]\n")
	}
	if hasBody {
		b.WriteString("      requestBody:\n")
		b.WriteString("        required: true\n")
		b.WriteString("        content:\n")
		b.WriteString("          application/json:\n")
		b.WriteString(fmt.Sprintf("            schema: { $ref: '#/components/schemas/%s' }\n", schema))
	}
	b.WriteString("      responses:\n")
	b.WriteString("        '200':\n")
	b.WriteString("          description: OK\n")
	b.WriteString("          content:\n")
	b.WriteString("            application/json:\n")
	b.WriteString(fmt.Sprintf("              schema: { $ref: '#/components/schemas/%s' }\n", schema))
}

func openAPIType(t string) string {
	switch t {
	case "int", "integer", "bigint":
		return "integer"
	case "float", "double", "decimal":
		return "number"
	case "bool", "boolean":
		return "boolean"
	default:
		return "string"
	}
}
