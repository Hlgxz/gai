**English** | [中文](README_zh.md)

# Gai - AI-Native Go Web Framework

> **Define once, generate everything.**

Gai is an AI-native full-stack Go web framework that blends Go's simplicity with Laravel's elegance. Describe your business with declarative YAML schemas — the framework generates APIs, database migrations, validation rules, and more automatically.

## Features

- **AI-Native Development** — Built-in rules for all major AI coding tools; every generated project is AI-ready out of the box
- **Schema-Driven** — Define models in YAML, auto-generate Model, Controller, Migration, and Routes
- **Elegant API** — Go-idiomatic core with Laravel-inspired chain calls and expressive routing
- **Service Container** — Laravel-style DI container with Singleton, Bind, and ServiceProvider
- **Multi-DB ORM** — Generic chainable query builder supporting MySQL, PostgreSQL, and SQLite
- **Database Migrations** — Versioned migration system with Up/Down/Rollback
- **Auth System** — Multi-guard design with built-in JWT + WeChat mini-program authentication
- **Mini-Program SDK** — WeChat / Alipay login, payment, and push notifications
- **Request Validation** — Laravel-style pipe rules (`required|email|min:5`)
- **Built-in Middleware** — CORS, Logger, Recovery, RateLimit ready to go
- **CLI Toolkit** — `gai` command-line tool for scaffolding and code generation

## AI-Native Development

Gai is the first Go framework with built-in rule files for every major AI coding tool. Every project created with `gai new` automatically includes AI rules, so any AI assistant can understand the framework and write correct code from the start.

### Supported AI Tools

| AI Tool | Rule File |
|---------|-----------|
| **Cursor** | `.cursor/rules/gai.mdc` |
| **Claude Code** | `CLAUDE.md` |
| **GitHub Copilot** | `.github/copilot-instructions.md` |
| **Windsurf** | `.windsurfrules` |
| **Kiro** | `.kiro/rules.md` |
| **Gemini / Qoder** | `.gemini/style-guide.md` |
| **Roo Code / Cline** | `.roo/rules.md` + `.clinerules` |
| **Augment / Antigravity** | `.augment/rules.md` |
| **Codex CLI** | `AGENTS.md` |

### Get Started in One Step

**Send this to any AI coding assistant:**

```
Create a project using the Gai framework (https://github.com/Hlgxz/gai)
```

The AI will automatically:
1. `go install github.com/Hlgxz/gai/cmd/gai@latest`
2. `gai new myapp --module github.com/user/myapp`
3. `cd myapp && go mod tidy`
4. The project already contains the AI tool's rule files — start coding immediately

## Quick Start

### Install CLI

```bash
go install github.com/Hlgxz/gai/cmd/gai@latest
```

### Create a Project

```bash
gai new myproject
cd myproject
go mod tidy
gai serve
```

### Project Structure

```
myproject/
├── app/
│   ├── controllers/     # HTTP controllers
│   └── models/          # ORM models
├── config/
│   └── app.yaml         # Configuration (supports ${ENV_VAR:default})
├── database/
│   └── migrations/      # Database migrations
├── routes/
│   └── routes.go        # Route registration
├── schemas/             # YAML schema definitions
├── storage/             # Logs, SQLite DB, uploads
├── .env                 # Environment variables
├── main.go              # Entry point
└── go.mod
```

## Core Usage

### Routing

```go
app := gai.New()
r := app.Router()

r.Get("/", homeHandler)
r.Post("/login", loginHandler)

r.Group("/api/v1", func(g *router.Group) {
    g.Use(middleware.Auth("jwt"))
    g.Get("/users", userCtrl.Index)
    g.Post("/users", userCtrl.Store)
    g.Get("/users/:id", userCtrl.Show)
    g.Resource("/posts", postCtrl)  // RESTful resource routes
})
```

### ORM Queries

```go
// Chain queries
users, err := orm.Get[User](
    orm.Query[User](db).
        Where("age", ">", 18).
        Where("status", "=", "active").
        OrderBy("created_at", "DESC").
        Limit(20),
)

// Pagination
page, err := orm.Paginate[User](
    orm.Query[User](db).Where("status", "=", "active"),
    1, 20,
)

// Create
user, err := orm.Create[User](db, &User{Name: "John", Email: "john@example.com"})
```

### Schema-Driven Development

Create `schemas/user.yaml`:

```yaml
model: User
table: users
fields:
  - name: name
    type: string
    size: 100
    rules: required|min:2|max:50
  - name: email
    type: string
    unique: true
    rules: required|email
  - name: phone
    type: string
    size: 20
    rules: phone
  - name: status
    type: enum
    values: [active, inactive, banned]
    default: "'active'"

api:
  prefix: /api/v1/users
  actions: [index, show, store, update, destroy]
  auth: jwt

relations:
  - type: hasMany
    model: Post
```

Generate code:

```bash
gai generate --schema schemas/user.yaml
```

Auto-generated files:
- `app/models/user.go`
- `app/controllers/user_controller.go`
- `database/migrations/xxx_create_users_table.go`
- `routes/user_routes.go`

### Authentication

```go
// JWT
guard := auth.NewJWTGuard("your-secret", 7200)
token, _ := guard.IssueToken(userID, nil)

// Protect routes with middleware
r.Group("/api", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    // ...
})
```

### WeChat Mini-Program

```go
wc := wechat.NewClient(wechat.Config{
    AppID:     "your-app-id",
    AppSecret: "your-app-secret",
})

// Login
session, err := wc.Auth().Code2Session(code)

// Payment
result, err := wc.Pay().UnifiedOrder(&wechat.Order{
    Body:       "Product description",
    OutTradeNo: "order-001",
    TotalFee:   100,
    OpenID:     session.OpenID,
})

// Subscribe message
err = wc.Message().SendSubscribe(&wechat.SubscribeMessage{
    ToUser:     openid,
    TemplateID: "tpl-id",
    Data:       map[string]wechat.MsgVal{"thing1": {Value: "Order shipped"}},
})
```

### Request Validation

```go
validator := ghttp.NewValidator(input, map[string]string{
    "name":  "required|min:2|max:50",
    "email": "required|email",
    "phone": "phone",
    "age":   "numeric|min:0|max:150",
})
if errs := validator.Validate(); errs != nil {
    c.JSON(422, errs)
    return
}
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gai new <name>` | Create a new project |
| `gai serve` | Start development server |
| `gai serve -w` | Watch mode (auto-restart) |
| `gai make model <Name>` | Generate a model |
| `gai make controller <Name>` | Generate a controller |
| `gai make middleware <Name>` | Generate middleware |
| `gai make migration <name>` | Generate a migration file |
| `gai generate --schema <path>` | Generate code from schema |
| `gai migrate` | Run database migrations |
| `gai migrate rollback` | Rollback last migration batch |
| `gai migrate status` | Show migration status |

## Tech Stack

- **HTTP**: Built on `net/http` — zero third-party router dependencies
- **Database**: `database/sql` + MySQL / PostgreSQL / SQLite drivers
- **Config**: YAML + `.env` environment variables
- **JWT**: `github.com/golang-jwt/jwt/v5`
- **CLI**: `github.com/spf13/cobra`
- **Logging**: `log/slog` (Go standard library)

## License

MIT License
