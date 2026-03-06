# CLAUDE.md — Gai Framework Guide for Claude Code

## What Is This Repository?

**Gai** is an AI-native Go web framework (`github.com/Hlgxz/gai`). It combines Go's simplicity with Laravel's elegance, featuring schema-driven code generation, a DI container, multi-database ORM, JWT/WeChat auth, and a full CLI toolkit.

## Quick Setup

```bash
go mod tidy && go build ./... && go run ./cmd/gai --help
```

## Architecture

The root package `gai` exports `Application`, `Container`, `Make[T]()`, and `ServiceProvider`. All HTTP handlers use `ghttp.Context` (import `github.com/Hlgxz/gai/http` as `ghttp`).

| Package | Purpose |
|---------|---------|
| `gai` (root) | Application kernel, DI container, service provider interface |
| `config/` | YAML config + .env loading, dot-notation Get/Set |
| `router/` | HTTP router with `:param` paths, groups, `Resource()` for RESTful |
| `http/` | `Context` (request/response), `Validator` (pipe rules) |
| `database/orm/` | Generic `Query[T]`, `Get[T]`, `First[T]`, `Create[T]`, `Paginate[T]` |
| `database/driver/` | `Driver` interface + MySQL, PostgreSQL, SQLite implementations |
| `database/migration/` | `Migrator` + `Blueprint` schema builder |
| `auth/` | `Manager` with multi-guard support, `JWTGuard` |
| `miniapp/wechat/` | Code2Session, UnifiedOrder (pay), SubscribeMessage |
| `miniapp/alipay/` | OAuth login |
| `ai/schema/` | YAML schema parser (`ParseFile`, `ParseDir`) |
| `ai/generator/` | Code generation: Model, Controller, Migration, Routes |
| `middleware/` | CORS, Logger, Recovery, RateLimit |
| `support/` | `Snake()`, `Camel()`, `Plural()`, `Hash()`, `Env()` |
| `cmd/gai/` | CLI: `new`, `serve`, `make`, `generate`, `migrate` |

## Code Conventions

- **Always** alias `github.com/Hlgxz/gai/http` as `ghttp` to avoid conflict with `net/http`.
- Handler signature: `func(c *ghttp.Context)`.
- Success: `c.Success(data)` → `{code: 0, message: "ok", data}`.
- Error: `c.Error(httpCode, message)` → `{code, message}`.
- ORM models embed `orm.Model` and use `gai:"..."` struct tags.
- Table names auto-derive: `UserPost` → `user_posts`.
- Files use snake_case: `user_controller.go`.
- No `init()` functions except in `database/driver/*.go` for driver registration.
- Middleware must call `c.Next()` to continue the chain.

## Schema-Driven Workflow

Create YAML schemas in `schemas/` or `examples/`, then generate:

```bash
go run ./cmd/gai generate --schema examples/user.yaml
```

Schema format:
```yaml
model: User
table: users
fields:
  - name: email
    type: string
    unique: true
    rules: required|email
api:
  prefix: /api/v1/users
  actions: [index, show, store, update, destroy]
  auth: jwt
relations:
  - type: hasMany
    model: Post
```

## Adding New Components

- **Model**: Embed `orm.Model`, add `gai` tags, place in `database/orm/` or user's `app/models/`.
- **Route**: Use `r.Get()`, `r.Group()`, `r.Resource()` on `*router.Router`.
- **Middleware**: Return `ghttp.HandlerFunc`, call `c.Next()` inside.
- **Guard**: Implement `auth.Guard` interface (Name, User, Check, Attempt, Logout).
- **Driver**: Implement `driver.Driver` interface, register via `driver.Register()` in `init()`.
- **CLI command**: Add cobra command in `cmd/gai/`, register in `main.go`.
- **Schema type**: Update `ai/schema/types.go` GoType + NeedsImport, plus all 3 driver ColumnType methods.

## Testing

```bash
go build ./...   # compile check
go vet ./...     # static analysis
go test ./...    # run tests
```

## Dependencies (do not change versions without reason)

- `gopkg.in/yaml.v3` — config/schema parsing
- `github.com/golang-jwt/jwt/v5` — JWT auth
- `github.com/spf13/cobra` — CLI
- `github.com/go-sql-driver/mysql`, `github.com/lib/pq`, `modernc.org/sqlite` — DB drivers
- `golang.org/x/crypto` — bcrypt
