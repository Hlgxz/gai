# Roo Code Rules — Gai Framework Source

Gai (`github.com/Hlgxz/gai`) — Go web framework **library**.

Usage: `gai new myapp` → generates project + AI rules for all tools.

## Conventions
- `import ghttp "github.com/Hlgxz/gai/http"` (mandatory)
- Handler: `func(c *ghttp.Context)` — `c.Success()`, `c.Error()`
- ORM: `orm.Query[T](db)`, `orm.Get[T]()`, `orm.Create[T]()`
- Model: embed `orm.Model`, `gai:"..."` tags
- Route: `:param`, `Group()`, `Resource()`
- Middleware: `c.Next()` required
- Validate: pipe syntax `required|email|min:5`
- Schema YAML → `gai generate`
