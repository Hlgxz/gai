# GitHub Copilot Instructions — Gai Framework

## Context

You are working in the **Gai** framework repository (`github.com/Hlgxz/gai`), an AI-native Go web framework for building mini-program backends and web services. It features a DI container, custom router, generic ORM, JWT auth, WeChat/Alipay SDKs, and schema-driven code generation.

## Setup

```bash
go mod tidy && go build ./...
```

## Code Style Requirements

1. **Always** import `github.com/Hlgxz/gai/http` with alias `ghttp`:
   ```go
   import ghttp "github.com/Hlgxz/gai/http"
   ```

2. Handler functions use `func(c *ghttp.Context)`, never raw `net/http` handlers.

3. Response patterns:
   - Success: `c.Success(data)` → `{"code": 0, "message": "ok", "data": ...}`
   - Error: `c.Error(code, message)` → `{"code": N, "message": "..."}`
   - JSON: `c.JSON(statusCode, obj)` for custom responses

4. ORM uses Go generics — always use package-level functions:
   ```go
   orm.Get[User](query)     // not query.Get()
   orm.First[User](query)   // returns (*User, error)
   orm.Create[User](db, m)  // returns (*User, error)
   orm.Paginate[User](q, page, perPage)
   ```

5. Models embed `orm.Model` and use `gai:"..."` struct tags:
   ```go
   type User struct {
       orm.Model
       Name string `json:"name" gai:"column:name;size:100;index"`
   }
   ```

6. Routes use `:param` syntax for path parameters:
   ```go
   r.Get("/users/:id", handler)  // c.Param("id") in handler
   ```

7. Middleware must call `c.Next()` to continue the handler chain.

8. File naming: `snake_case.go` for all Go source files.

## Package Reference

| Import | Alias | Purpose |
|--------|-------|---------|
| `github.com/Hlgxz/gai` | `gai` | Application, Container, Make[T] |
| `github.com/Hlgxz/gai/http` | `ghttp` | Context, HandlerFunc, Validator |
| `github.com/Hlgxz/gai/router` | `router` | Router, Group, ResourceController |
| `github.com/Hlgxz/gai/database/orm` | `orm` | DB, Model, Query, Get, First, Create |
| `github.com/Hlgxz/gai/database/driver` | `driver` | Driver interface, Register |
| `github.com/Hlgxz/gai/database/migration` | `migration` | Migrator, Blueprint, Migration |
| `github.com/Hlgxz/gai/auth` | `auth` | Manager, Guard, JWTGuard |
| `github.com/Hlgxz/gai/middleware` | `middleware` | CORS, Logger, Recovery, RateLimit |
| `github.com/Hlgxz/gai/miniapp/wechat` | `wechat` | Client, Auth, Pay, Message |
| `github.com/Hlgxz/gai/miniapp/alipay` | `alipay` | Client, Auth |
| `github.com/Hlgxz/gai/ai/schema` | `schema` | ParseFile, ParseDir, Schema |
| `github.com/Hlgxz/gai/ai/generator` | `generator` | Generator, GenerateAll |
| `github.com/Hlgxz/gai/support` | `support` | Snake, Camel, Plural, Hash, Env |

## Schema-Driven Development

Create YAML in `schemas/`, run `go run ./cmd/gai generate --schema schemas/`. Each schema auto-generates model, controller, migration, and route files.

## Validation Rules

Pipe-separated: `required|email|min:2|max:100|numeric|phone|alpha|alphanumeric|url|in:a,b,c|regex:pattern`.
