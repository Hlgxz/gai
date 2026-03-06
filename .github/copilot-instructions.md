# GitHub Copilot — Gai Framework Source

This is the **source repository** of the Gai Go web framework (`github.com/Hlgxz/gai`).

Users consume this as a library:
```bash
go install github.com/Hlgxz/gai/cmd/gai@latest
gai new myapp --module github.com/user/myapp
```

`gai new` auto-generates AI rules files for all major tools (Cursor, Claude, Copilot, Windsurf, Kiro, Gemini, Roo Code, Augment).

## Key Conventions

- Import `github.com/Hlgxz/gai/http` as `ghttp` (mandatory alias)
- Handler: `func(c *ghttp.Context)` — use `c.Success()`, `c.Error()`, `c.JSON()`
- ORM generics: `orm.Query[T](db)`, `orm.Get[T](q)`, `orm.Create[T](db, &item)`
- Models embed `orm.Model` with `gai:"..."` struct tags
- Router: `r.Get("/:param", h)`, `r.Group()`, `r.Resource()`
- Middleware: return `ghttp.HandlerFunc`, must call `c.Next()`
- Validation: `ghttp.NewValidator(data, rules)` with pipe syntax `required|email|min:5`
- Schema-driven: YAML in `schemas/` → `gai generate --schema schemas/`

## Package Map

| Import | Alias | Exports |
|--------|-------|---------|
| `github.com/Hlgxz/gai` | `gai` | Application, Container, Make[T] |
| `github.com/Hlgxz/gai/http` | `ghttp` | Context, HandlerFunc, Validator |
| `github.com/Hlgxz/gai/router` | `router` | Router, Group, ResourceController |
| `github.com/Hlgxz/gai/database/orm` | `orm` | DB, Model, Query[T], Get[T], Create[T] |
| `github.com/Hlgxz/gai/auth` | `auth` | Manager, Guard, JWTGuard |
| `github.com/Hlgxz/gai/middleware` | `middleware` | CORS(), Logger(), Recovery(), RateLimit() |
| `github.com/Hlgxz/gai/miniapp/wechat` | `wechat` | Client, Auth, Pay, Message |
| `github.com/Hlgxz/gai/support` | `support` | Snake, Camel, Hash, Env |
