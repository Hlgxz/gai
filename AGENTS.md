# AGENTS.md — Gai Framework Instructions for AI Coding Agents

> This file provides context for AI coding agents (Codex CLI, OpenAI Codex, and others).

## Repository

- **Name**: Gai — AI-native Go web framework
- **URL**: https://github.com/Hlgxz/gai
- **Module**: `github.com/Hlgxz/gai`
- **Language**: Go 1.22+
- **Motto**: "Define once, generate everything."

## One-Step Environment Setup

```bash
git clone https://github.com/Hlgxz/gai.git && cd gai && go mod tidy && go build ./...
```

Verify: `go run ./cmd/gai --help` should show available commands.

## Architecture Summary

Gai is a full-stack Go web framework with these layers:

1. **Application Layer** (`app.go`, `container.go`, `provider.go`): DI container with Bind/Singleton/Make[T], ServiceProvider pattern for modular boot.
2. **HTTP Layer** (`router/`, `http/`, `middleware/`): Custom router with path params (`:id`), groups, RESTful resources. Context wraps request/response with JSON/HTML helpers.
3. **Data Layer** (`database/`): Multi-driver ORM with generic query builder, migration engine with Blueprint schema builder.
4. **Auth Layer** (`auth/`): Multi-guard system supporting JWT and extensible to WeChat/Alipay.
5. **Mini-program Layer** (`miniapp/`): WeChat (login, pay, messages) and Alipay (login) SDKs.
6. **AI/Generation Layer** (`ai/`): YAML schema parser + code generator producing Model, Controller, Migration, Routes.
7. **CLI** (`cmd/gai/`): Project scaffolding, code generation, dev server, migrations.

## Critical Convention: HTTP Package Aliasing

The framework has its own `http` package. **Always** import it as `ghttp`:

```go
import ghttp "github.com/Hlgxz/gai/http"
```

Handler functions use `func(c *ghttp.Context)`, not `http.HandlerFunc`.

## Key Patterns

### Route Registration
```go
r := app.Router()
r.Get("/", handler)
r.Group("/api", func(g *router.Group) {
    g.Use(middleware)
    g.Resource("/users", controller)  // auto CRUD routes
})
```

### ORM Queries (always use generic functions)
```go
items, _ := orm.Get[Item](orm.Query[Item](db).Where("active", "=", true).OrderBy("id", "DESC"))
item, _ := orm.First[Item](orm.Query[Item](db).Where("id", "=", 1))
page, _ := orm.Paginate[Item](orm.Query[Item](db), pageNum, perPage)
created, _ := orm.Create[Item](db, &Item{Name: "x"})
orm.Update[Item](db, item)
orm.Delete[Item](db, item)
```

### Model Definition
```go
type Item struct {
    orm.Model                                    // ID, CreatedAt, UpdatedAt, DeletedAt
    Name string `json:"name" gai:"column:name;size:100"`
}
```

### Validation
```go
v := ghttp.NewValidator(data, map[string]string{
    "email": "required|email",
    "age":   "required|numeric|min:0|max:150",
})
if errs := v.Validate(); errs != nil { /* handle */ }
```

### Schema-Driven Code Generation
```bash
# Generate from single schema
go run ./cmd/gai generate --schema schemas/product.yaml

# Generate from all schemas in directory
go run ./cmd/gai generate --schema schemas/
```

## File Naming Rules

- Go files: `snake_case.go` (e.g., `user_controller.go`)
- Schema files: `snake_case.yaml`
- Packages: short, lowercase, single word
- Models: PascalCase struct names → auto snake_case+plural table names

## Do NOT

- Use `init()` outside of `database/driver/*.go`
- Import `github.com/Hlgxz/gai/http` without aliasing as `ghttp`
- Use raw `http.ResponseWriter`/`http.Request` in handlers — use `*ghttp.Context`
- Add third-party router libraries (gin, echo, chi) — Gai has its own router
- Hardcode database SQL — use the query builder or migration blueprint
