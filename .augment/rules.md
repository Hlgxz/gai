# Augment / Antigravity Rules — Gai Framework Source

Gai (`github.com/Hlgxz/gai`) — Go web framework **library**.

Usage: `gai new myapp` → generates project + AI rules for all tools.

## Conventions
- `import ghttp "github.com/Hlgxz/gai/http"` (mandatory alias)
- Handler: `func(c *ghttp.Context)` — c.Success(), c.Error()
- ORM generics: `orm.Query[T]()`, `orm.Get[T]()`, `orm.Create[T]()`
- Model: embed `orm.Model`, `gai:"..."` struct tags
- Route: `:param`, `Group()`, `Resource()`
- Middleware: must call `c.Next()`
- Validate: pipe syntax `required|email|min:5`
- Schema YAML in `schemas/` → `gai generate`
- No gin/echo/chi — Gai has its own router
