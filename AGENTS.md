# AGENTS.md — Gai Framework

> 面向所有 AI 编码工具 (Codex CLI, Claude Code, Cursor, Copilot, Windsurf, Kiro, Gemini, Roo Code 等)

## 这是什么

Gai (`github.com/Hlgxz/gai`) 是一个 Go Web 框架**库**，用户通过 `go get` 导入到自己的业务项目中使用。

## 当用户说"用 Gai 框架开发"时，执行以下步骤

### 步骤 1：创建业务项目

```bash
# 安装 CLI（如果还没装）
go install github.com/Hlgxz/gai/cmd/gai@latest

# 创建新项目（自动生成完整结构 + AI 规则文件）
gai new myapp --module github.com/user/myapp
cd myapp
go mod tidy
```

### 步骤 2：立即可以开发

项目已包含：
- `main.go` — 入口，已配好 Gai 应用
- `routes/routes.go` — 路由注册
- `config/app.yaml` + `.env` — 配置
- `schemas/` — Schema 定义目录
- 全平台 AI 规则文件 — 所有 AI 工具即开即用

### 步骤 3：开发新功能

**Schema 驱动（推荐）**：
```bash
# 1. 创建 schemas/product.yaml
# 2. 生成代码
gai generate --schema schemas/product.yaml
# 3. 注册路由
# 4. 运行
gai serve
```

**手动开发**：
```bash
gai make model Product
gai make controller Product
gai make migration create_products_table
```

## Gai 框架核心规则

### Import 别名（强制）
```go
import ghttp "github.com/Hlgxz/gai/http"   // 必须
```

### Handler 签名
```go
func Handler(c *ghttp.Context) {
    c.Success(data)       // {"code":0,"message":"ok","data":...}
    c.Error(400, "msg")   // {"code":400,"message":"msg"}
}
```

### ORM 用法
```go
orm.Query[User](db).Where("status", "=", "active")
orm.Get[User](query)          // []User
orm.First[User](query)        // *User
orm.Create[User](db, &user)   // 插入
orm.Paginate[User](q, 1, 20)  // 分页
```

### Model 定义
```go
type User struct {
    orm.Model
    Name string `json:"name" gai:"column:name;size:100"`
}
```

### 路由
```go
r.Group("/api/v1", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    g.Resource("/users", ctrl) // 自动 5 条 CRUD 路由
})
```

### 校验
```go
ghttp.NewValidator(data, map[string]string{"email": "required|email"}).Validate()
```

## 包速查

| 导入 | 别名 | 关键导出 |
|------|------|---------|
| github.com/Hlgxz/gai | gai | Application, Container, Make[T] |
| github.com/Hlgxz/gai/http | ghttp | Context, HandlerFunc, Validator |
| github.com/Hlgxz/gai/router | router | Router, Group, ResourceController |
| github.com/Hlgxz/gai/database/orm | orm | DB, Model, Query[T], Get[T], Create[T] |
| github.com/Hlgxz/gai/auth | auth | Manager, Guard, JWTGuard |
| github.com/Hlgxz/gai/middleware | middleware | CORS(), Logger(), Recovery() |
| github.com/Hlgxz/gai/miniapp/wechat | wechat | Client, Auth, Pay, Message |

## 禁止

- 导入 gai/http 不加 ghttp 别名
- 使用 net/http 的 ResponseWriter 替代 ghttp.Context
- 中间件跳过 c.Next()
- 添加 gin/echo/chi（Gai 自带路由）
