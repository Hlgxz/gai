# Roo Code Rules — Gai Framework

## Repository

**Gai** — AI-native Go web framework (`github.com/Hlgxz/gai`)

## Quick Start

```bash
go mod tidy && go build ./... && go run ./cmd/gai --help
```

## Core Conventions

1. **Import alias**: `import ghttp "github.com/Hlgxz/gai/http"` — ALWAYS use `ghttp`.

2. **Handler**: `func(c *ghttp.Context)` — use context methods for input/output:
   - Input: `c.Param("id")`, `c.Query("key")`, `c.BindJSON(&dst)`, `c.PostForm("field")`
   - Output: `c.Success(data)`, `c.Error(code, msg)`, `c.JSON(status, obj)`, `c.HTML(status, html)`
   - Flow: `c.Next()` (continue chain), `c.Abort()` (stop chain)

3. **ORM generics**:
   ```go
   orm.Query[T](db)              // build query
   orm.Get[T](q)                 // → []T
   orm.First[T](q)               // → *T
   orm.Paginate[T](q, page, pp)  // → *Pagination[T]
   orm.Create[T](db, &t)         // insert
   orm.Update[T](db, &t)         // update
   orm.Delete[T](db, &t)         // delete/soft-delete
   ```

4. **Model pattern**:
   ```go
   type Item struct {
       orm.Model
       Name string `json:"name" gai:"column:name;size:100"`
   }
   ```

5. **Router**:
   ```go
   r.Group("/api", func(g *router.Group) {
       g.Use(authMw)
       g.Resource("/items", ctrl)
   })
   ```

6. **Middleware**: return `ghttp.HandlerFunc`, MUST call `c.Next()`.

7. **Validation**: `ghttp.NewValidator(data, map[string]string{"field": "required|email"})`.

8. **Schema-driven**: YAML in `schemas/` → `gai generate --schema schemas/` → full CRUD code.

9. **Files**: snake_case.go. **Tables**: auto plural snake_case.

10. **No init()** outside `database/driver/`. Use `ServiceProvider` pattern.

## Packages

| Package | Import | Key Exports |
|---------|--------|-------------|
| Root | `gai` | Application, Container, Make[T], ServiceProvider |
| HTTP | `ghttp` | Context, HandlerFunc, Validator |
| Router | `router` | Router, Group, ResourceController |
| ORM | `orm` | DB, Model, Query[T], Get[T], Create[T] |
| Auth | `auth` | Manager, Guard, JWTGuard |
| Middleware | `middleware` | CORS(), Logger(), Recovery(), RateLimit() |
| WeChat | `wechat` | Client, Auth, Pay, Message |
| Schema | `schema` | ParseFile, ParseDir |
| Generator | `generator` | Generator, GenerateAll |
| Support | `support` | Snake, Camel, Plural, Hash, Env |
