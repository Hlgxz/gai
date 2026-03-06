# Gemini Style Guide — Gai Framework

## About This Repository

**Gai** (`github.com/Hlgxz/gai`) is an AI-native Go web framework for building mini-program backends and web services. It features a Laravel-inspired DI container, custom HTTP router, generic ORM, JWT/WeChat authentication, and schema-driven code generation.

## Environment Setup (execute immediately)

```bash
go mod tidy && go build ./... && go run ./cmd/gai --help
```

## Project Structure

```
gai/                    → Root: Application, Container, ServiceProvider
├── cmd/gai/            → CLI tool (new, serve, make, generate, migrate)
├── config/             → YAML + .env config manager
├── router/             → HTTP router (groups, :param paths, Resource)
├── http/               → Context (request/response), Validator
├── database/orm/       → Generic ORM: Query[T], Get[T], Create[T], Paginate[T]
├── database/driver/    → Driver interface: MySQL, PostgreSQL, SQLite
├── database/migration/ → Migrator + Blueprint schema builder
├── auth/               → Multi-guard auth manager + JWT guard
├── miniapp/wechat/     → WeChat SDK: login, pay, subscribe messages
├── miniapp/alipay/     → Alipay SDK: OAuth login
├── ai/schema/          → YAML schema parser
├── ai/generator/       → Code generator (Model/Controller/Migration/Routes)
├── middleware/          → CORS, Logger, Recovery, RateLimit
├── support/            → String/hash/env utilities
└── examples/           → Sample schema YAML files
```

## Mandatory Rules

### 1. HTTP Package Import Alias
ALWAYS use `ghttp` alias to avoid collision with standard `net/http`:
```go
import ghttp "github.com/Hlgxz/gai/http"
```

### 2. Handler Functions
All HTTP handlers use `func(c *ghttp.Context)`. Never use `http.HandlerFunc`.

### 3. Response Envelope
- Success: `c.Success(data)` → `{"code": 0, "message": "ok", "data": ...}`
- Error: `c.Error(httpCode, message)` → `{"code": N, "message": "..."}`

### 4. ORM — Generic Package Functions
```go
query := orm.Query[User](db).Where("status", "=", "active")
users, err := orm.Get[User](query)
user, err := orm.First[User](query)
page, err := orm.Paginate[User](query, 1, 20)
created, err := orm.Create[User](db, &User{Name: "test"})
err = orm.Update[User](db, user)
err = orm.Delete[User](db, user)
```

### 5. Model Struct
```go
type User struct {
    orm.Model                                         // ID, CreatedAt, UpdatedAt, DeletedAt
    Name  string `json:"name"  gai:"column:name;size:100"`
    Email string `json:"email" gai:"column:email;unique"`
}
```

### 6. Routing
```go
r.Get("/path/:id", handler)
r.Group("/api", func(g *router.Group) {
    g.Use(middleware)
    g.Resource("/users", ctrl)  // GET, POST, GET/:id, PUT/:id, DELETE/:id
})
```

### 7. Validation
Pipe syntax: `required|email|min:2|max:100|numeric|phone|in:a,b,c`
```go
v := ghttp.NewValidator(data, rules)
errs := v.Validate() // nil if valid
```

### 8. File Naming
Go files: `snake_case.go`. Tables: auto snake_case plural of model name.

## Schema-Driven Code Generation

Define YAML in `schemas/`, generate with `go run ./cmd/gai generate --schema schemas/`. Produces complete CRUD: model, controller, migration, routes.

## Do Not

- Import `github.com/Hlgxz/gai/http` without `ghttp` alias
- Use `init()` outside database driver files
- Add external router libraries (gin, echo, chi)
- Use raw SQL when query builder or migration Blueprint suffices
- Skip `c.Next()` in middleware (breaks the handler chain)
