# Gai - AI 原生 Go Web 框架

> **Define once, generate everything.** — 定义一次，生成一切。

Gai 是一个 AI 原生的 Go Web 全栈框架，融合 Go 语言的简洁高效与 Laravel 的优雅设计。通过 Schema 驱动开发和 AI 代码生成，让你用声明式的方式定义业务，框架自动推导出 API、数据库迁移、校验规则等全部代码。

## 特性

- **Schema 驱动** — 用 YAML 描述业务模型，自动生成 Model、Controller、Migration、Routes
- **优雅的 API** — Go 惯用风格为主，关键处借鉴 Laravel 的链式调用和表达式路由
- **服务容器** — 类 Laravel 的依赖注入容器，支持 Singleton、Bind、ServiceProvider
- **多数据库 ORM** — 链式查询构建器，支持 MySQL、PostgreSQL、SQLite
- **数据库迁移** — 版本化的数据库迁移系统，支持 Up/Down/Rollback
- **认证系统** — 多 Guard 设计，内置 JWT + 微信小程序认证
- **小程序 SDK** — 微信/支付宝小程序登录、支付、消息推送
- **请求校验** — Laravel 风格的管道式校验规则 (required|email|min:5)
- **内置中间件** — CORS、Logger、Recovery、RateLimit 开箱即用
- **CLI 工具** — `gai` 命令行工具，项目脚手架、代码生成一键完成
- **AI 原生开发** — 内置全平台 AI Agent Rules，支持所有主流 AI 编程工具即开即用

## AI 原生开发支持

Gai 是首个为所有主流 AI 编码工具内置规则文件的 Go 框架。`gai new` 创建的每个业务项目都自动包含全平台 AI 规则文件，让任何 AI 编程工具打开项目就能立即理解框架并正确编码。

### 支持的 AI 工具

| AI 工具 | 规则文件 |
|---------|---------|
| **Cursor** | `.cursor/rules/gai.mdc` |
| **Claude Code** | `CLAUDE.md` |
| **GitHub Copilot** | `.github/copilot-instructions.md` |
| **Windsurf** | `.windsurfrules` |
| **Kiro** | `.kiro/rules.md` |
| **Gemini / Qoder** | `.gemini/style-guide.md` |
| **Roo Code / Cline** | `.roo/rules.md` + `.clinerules` |
| **Augment / Antigravity** | `.augment/rules.md` |
| **Codex CLI** | `AGENTS.md` |

### 一步开始

**给任何 AI 编程助手发送：**

```
请帮我用 Gai 框架 (https://github.com/Hlgxz/gai) 创建一个项目
```

AI 会自动执行：
1. `go install github.com/Hlgxz/gai/cmd/gai@latest`
2. `gai new myapp --module github.com/user/myapp`
3. `cd myapp && go mod tidy`
4. 项目已包含该 AI 工具的规则文件，立即可以理解框架约定并开始开发

## 快速开始

### 安装 CLI

```bash
go install github.com/Hlgxz/gai/cmd/gai@latest
```

### 创建项目

```bash
gai new myproject
cd myproject
go mod tidy
gai serve
```

### 项目结构

```
myproject/
├── app/
│   ├── controllers/     # 控制器
│   └── models/          # 模型
├── config/
│   └── app.yaml         # 配置文件
├── database/
│   └── migrations/      # 数据库迁移
├── routes/
│   └── routes.go        # 路由注册
├── schemas/             # Schema 定义文件
├── storage/             # 存储目录
├── .env                 # 环境变量
├── main.go              # 入口文件
└── go.mod
```

## 核心用法

### 路由

```go
app := gai.New()
r := app.Router()

r.Get("/", homeHandler)
r.Post("/login", loginHandler)

r.Group("/api/v1", func(g *router.Group) {
    g.Use(middleware.Auth("jwt"))
    g.Get("/users", userCtrl.Index)
    g.Post("/users", userCtrl.Store)
    g.Get("/users/:id", userCtrl.Show)
    g.Resource("/posts", postCtrl)
})
```

### ORM 查询

```go
// 链式查询
users, err := orm.Get[User](
    orm.Query[User](db).
        Where("age", ">", 18).
        Where("status", "=", "active").
        OrderBy("created_at", "DESC").
        Limit(20),
)

// 分页
page, err := orm.Paginate[User](
    orm.Query[User](db).Where("status", "=", "active"),
    1, 20,
)

// 创建
user, err := orm.Create[User](db, &User{Name: "张三", Email: "z@test.com"})
```

### Schema 驱动开发

创建 `schemas/user.yaml`:

```yaml
model: User
table: users
fields:
  - name: name
    type: string
    size: 100
    rules: required|min:2|max:50
  - name: email
    type: string
    unique: true
    rules: required|email
  - name: phone
    type: string
    size: 20
    rules: phone
  - name: status
    type: enum
    values: [active, inactive, banned]
    default: "'active'"

api:
  prefix: /api/v1/users
  actions: [index, show, store, update, destroy]
  auth: jwt

relations:
  - type: hasMany
    model: Post
```

生成代码:

```bash
gai generate --schema schemas/user.yaml
```

自动产出:
- `app/models/user.go`
- `app/controllers/user_controller.go`
- `database/migrations/xxx_create_users_table.go`
- `routes/user_routes.go`

### 认证

```go
// JWT
guard := auth.NewJWTGuard("your-secret", 7200)
token, _ := guard.IssueToken(userID, nil)

// 中间件保护路由
r.Group("/api", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    // ...
})
```

### 微信小程序

```go
wc := wechat.NewClient(wechat.Config{
    AppID:     "your-app-id",
    AppSecret: "your-app-secret",
})

// 登录
session, err := wc.Auth().Code2Session(code)

// 支付
result, err := wc.Pay().UnifiedOrder(&wechat.Order{
    Body:       "商品描述",
    OutTradeNo: "order-001",
    TotalFee:   100,
    OpenID:     session.OpenID,
})

// 订阅消息
err = wc.Message().SendSubscribe(&wechat.SubscribeMessage{
    ToUser:     openid,
    TemplateID: "tpl-id",
    Data:       map[string]wechat.MsgVal{"thing1": {Value: "订单已发货"}},
})
```

### 请求校验

```go
validator := ghttp.NewValidator(input, map[string]string{
    "name":  "required|min:2|max:50",
    "email": "required|email",
    "phone": "phone",
    "age":   "numeric|min:0|max:150",
})
if errs := validator.Validate(); errs != nil {
    c.JSON(422, errs)
    return
}
```

## CLI 命令

| 命令 | 说明 |
|------|------|
| `gai new <name>` | 创建新项目 |
| `gai serve` | 启动开发服务器 |
| `gai serve -w` | 热重载模式 |
| `gai make model <Name>` | 生成模型 |
| `gai make controller <Name>` | 生成控制器 |
| `gai make middleware <Name>` | 生成中间件 |
| `gai make migration <name>` | 生成迁移文件 |
| `gai generate --schema <path>` | 从 Schema 生成代码 |
| `gai migrate` | 执行数据库迁移 |
| `gai migrate rollback` | 回滚迁移 |
| `gai migrate status` | 查看迁移状态 |

## 技术栈

- **HTTP**: 基于 `net/http`，零第三方路由依赖
- **数据库**: `database/sql` + MySQL / PostgreSQL / SQLite 驱动
- **配置**: YAML + .env 环境变量
- **JWT**: `github.com/golang-jwt/jwt/v5`
- **CLI**: `github.com/spf13/cobra`
- **日志**: `log/slog` (Go 标准库)

## 许可证

MIT License
