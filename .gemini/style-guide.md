# Gemini Style Guide — Gai Framework Source

Gai (`github.com/Hlgxz/gai`) is a Go web framework **library**.

## How Users Use This Framework
```bash
go install github.com/Hlgxz/gai/cmd/gai@latest
gai new myapp --module github.com/user/myapp
cd myapp && go mod tidy && gai serve
```

`gai new` generates a complete project with AI rules for all major tools.

## Mandatory Conventions

1. **Import alias**: `import ghttp "github.com/Hlgxz/gai/http"`
2. **Handler**: `func(c *ghttp.Context)` — never raw net/http
3. **Response**: `c.Success(data)`, `c.Error(code, msg)`
4. **ORM**: `orm.Query[T](db)`, `orm.Get[T](q)`, `orm.Create[T](db, &t)`
5. **Model**: embed `orm.Model`, `gai:"..."` tags
6. **Middleware**: must call `c.Next()`
7. **Files**: snake_case.go
8. **No** gin/echo/chi — Gai has its own router
