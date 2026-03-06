# Kiro Rules — Gai Framework Source

Gai (`github.com/Hlgxz/gai`) is a Go web framework **library**, not an application.

## Usage
```bash
go install github.com/Hlgxz/gai/cmd/gai@latest
gai new myapp --module github.com/user/myapp  # generates project + AI rules
cd myapp && go mod tidy && gai serve
```

## Conventions
- Import `gai/http` as `ghttp` (mandatory)
- Handler: `func(c *ghttp.Context)`
- ORM: `orm.Query[T](db)`, `orm.Get[T]()`, `orm.Create[T]()`, `orm.Paginate[T]()`
- Model: embed `orm.Model`, `gai:"..."` tags
- Route: `r.Group()`, `r.Resource()`, `:param` paths
- Middleware: `ghttp.HandlerFunc`, must `c.Next()`
- Validate: `ghttp.NewValidator(data, rules)`, pipe syntax
- Schema: YAML → `gai generate --schema`

## Quick API Reference

| Operation | Code |
|-----------|------|
| New app | `gai.New()` |
| Load config | `app.LoadConfig("config")` |
| Defaults | `app.UseDefaults()` |
| DI resolve | `gai.Make[T](container, "name")` |
| Query | `orm.Query[T](db).Where(...)` |
| Get all | `orm.Get[T](query)` |
| Get one | `orm.First[T](query)` |
| Paginate | `orm.Paginate[T](query, page, pp)` |
| Create | `orm.Create[T](db, &item)` |
| JWT | `guard.IssueToken(uid, extra)` |
| WeChat | `client.Auth().Code2Session(code)` |
