# Augment / Antigravity Rules — Gai Framework

## Repository

**Gai** (`github.com/Hlgxz/gai`) — AI-native Go web framework for mini-programs and web services.

## Setup

```bash
go mod tidy && go build ./... && go run ./cmd/gai --help
```

## Essential Conventions

### Import Alias (mandatory)
```go
import ghttp "github.com/Hlgxz/gai/http"
```

### Handler Functions
```go
func Handler(c *ghttp.Context) {
    c.Success(data)  // {"code":0,"message":"ok","data":...}
    c.Error(400, "msg") // {"code":400,"message":"msg"}
}
```

### ORM (generic functions)
```go
orm.Query[User](db).Where("active", "=", true).OrderBy("id", "DESC")
orm.Get[User](query)           // []User
orm.First[User](query)         // *User
orm.Create[User](db, &user)    // insert
orm.Paginate[User](query, 1, 20) // paginated
```

### Models
```go
type User struct {
    orm.Model
    Name string `json:"name" gai:"column:name;size:100;unique"`
}
```

### Routing
```go
r.Group("/api/v1", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    g.Get("/users", ctrl.Index)
    g.Resource("/posts", postCtrl)
})
```

### Validation
```go
ghttp.NewValidator(data, map[string]string{
    "email": "required|email",
    "name":  "required|min:2|max:50",
}).Validate()
```

### Schema-Driven
Create YAML in `schemas/`, run `gai generate --schema schemas/` to auto-generate Model + Controller + Migration + Routes.

## Architecture

| Layer | Packages | Key Types |
|-------|----------|-----------|
| Core | `gai` | Application, Container, Make[T] |
| HTTP | `http`, `router`, `middleware` | Context, Router, Group, CORS/Logger/Recovery |
| Data | `database/orm`, `database/driver`, `database/migration` | DB, Model, Query[T], Driver, Migrator, Blueprint |
| Auth | `auth` | Manager, Guard, JWTGuard, Claims |
| MiniApp | `miniapp/wechat`, `miniapp/alipay` | Client, Auth, Pay, Message |
| AI | `ai/schema`, `ai/generator` | Schema, Generator |
| CLI | `cmd/gai` | new, serve, make, generate, migrate |
| Utils | `support` | Snake, Camel, Plural, Hash, Env |

## Forbidden Patterns

- Importing `gai/http` without alias → causes `net/http` collision
- Using `init()` outside driver registration
- Adding gin/echo/chi dependencies
- Skipping `c.Next()` in middleware
- Raw SQL when query builder works
