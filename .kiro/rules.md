# Kiro Rules — Gai Framework

## Project Overview

- **Name**: Gai (Go + AI)
- **Type**: AI-native Go web framework
- **Module**: `github.com/Hlgxz/gai`
- **Purpose**: Build mini-program backends and web services with schema-driven code generation

## Setup Command

```bash
go mod tidy && go build ./... && go run ./cmd/gai --help
```

## Architectural Layers

1. **Core** (`app.go`, `container.go`, `provider.go`) — DI container, application lifecycle
2. **HTTP** (`router/`, `http/`, `middleware/`) — routing, context, built-in middleware
3. **Data** (`database/`) — ORM with generics, multi-DB drivers, migrations
4. **Auth** (`auth/`) — multi-guard JWT authentication
5. **MiniApp** (`miniapp/`) — WeChat and Alipay SDKs
6. **AI** (`ai/`) — YAML schema parser + code generator
7. **CLI** (`cmd/gai/`) — scaffolding and tooling

## Coding Standards

### Import Convention (CRITICAL)
```go
import ghttp "github.com/Hlgxz/gai/http"  // ALWAYS alias as ghttp
```

### Handler Pattern
```go
func MyHandler(c *ghttp.Context) {
    id := c.ParamInt("id")
    data, err := orm.First[MyModel](orm.Query[MyModel](db).Where("id", "=", id))
    if err != nil {
        c.Error(500, err.Error())
        return
    }
    c.Success(data)
}
```

### Model Pattern
```go
type MyModel struct {
    orm.Model
    Field string `json:"field" gai:"column:field;size:100;index"`
}
```

### Route Pattern
```go
r.Group("/api/v1", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    g.Resource("/items", itemController)
})
```

### Middleware Pattern
```go
func MyMiddleware() ghttp.HandlerFunc {
    return func(c *ghttp.Context) {
        // pre-processing
        c.Next()
        // post-processing
    }
}
```

### Validation Pattern
```go
v := ghttp.NewValidator(input, map[string]string{
    "name":  "required|min:2|max:50",
    "email": "required|email",
})
if errs := v.Validate(); errs != nil {
    c.JSON(422, map[string]any{"code": 422, "errors": errs})
    return
}
```

## Schema-Driven Workflow

1. Define YAML schema in `schemas/`
2. Run `go run ./cmd/gai generate --schema schemas/`
3. Generated: Model + Controller + Migration + Routes
4. Register routes, run migrations, done

## File Naming

- Go source: `snake_case.go`
- Schema: `snake_case.yaml`
- DB tables: auto `snake_case` plural (`UserPost` → `user_posts`)

## Key APIs Quick Reference

| Operation | Code |
|-----------|------|
| Create app | `gai.New()` |
| Load config | `app.LoadConfig("config")` |
| Get router | `app.Router()` |
| DI resolve | `gai.Make[T](container, "name")` |
| Query | `orm.Query[T](db).Where(...).OrderBy(...)` |
| Get all | `orm.Get[T](query)` |
| Get one | `orm.First[T](query)` |
| Paginate | `orm.Paginate[T](query, page, perPage)` |
| Create | `orm.Create[T](db, &item)` |
| JWT token | `guard.IssueToken(userID, extra)` |
| WeChat login | `wechatClient.Auth().Code2Session(code)` |
| Validate | `ghttp.NewValidator(data, rules).Validate()` |
