# CLAUDE.md — Gai Framework

## 这是什么

Gai (`github.com/Hlgxz/gai`) 是一个 AI 原生的 Go Web 框架，作为 **库** 被业务项目导入使用。

## 用户如何使用此框架

```bash
# 方式一：通过 CLI 创建项目（推荐）
go install github.com/Hlgxz/gai/cmd/gai@latest
gai new myapp --module github.com/user/myapp
cd myapp && go mod tidy && gai serve

# 方式二：手动创建项目
mkdir myapp && cd myapp
go mod init github.com/user/myapp
go get github.com/Hlgxz/gai
# 然后编写 main.go 导入 gai
```

`gai new` 会自动生成完整项目结构，包含所有 AI 工具的规则文件（.cursor/rules/、CLAUDE.md、AGENTS.md 等），让 AI 编程工具能立即理解框架并开始开发。

## 框架架构

| 包 | 用途 |
|----|------|
| `gai` (根包) | Application Kernel, DI Container, ServiceProvider |
| `config/` | YAML 配置 + .env 加载 |
| `router/` | HTTP 路由 (:param、Group、Resource) |
| `http/` | Context (请求/响应)、Validator (校验器) |
| `database/orm/` | 泛型 ORM: Query[T], Get[T], Create[T], Paginate[T] |
| `database/driver/` | 数据库驱动抽象 (MySQL/PG/SQLite) |
| `database/migration/` | 迁移引擎 + Blueprint Schema Builder |
| `auth/` | 多 Guard 认证 + JWT |
| `miniapp/wechat/` | 微信小程序 SDK |
| `miniapp/alipay/` | 支付宝小程序 SDK |
| `ai/schema/` | YAML Schema 解析器 |
| `ai/generator/` | 代码生成 (Model/Controller/Migration/Routes) |
| `middleware/` | CORS, Logger, Recovery, RateLimit |
| `support/` | 工具函数 (Snake, Camel, Hash, Env) |
| `cmd/gai/` | CLI (new, serve, make, generate, migrate) |

## 框架开发约定

- `github.com/Hlgxz/gai/http` **必须** 使用 `ghttp` 别名导入
- Handler 签名: `func(c *ghttp.Context)`
- ORM 使用包级泛型函数: `orm.Get[T]()`, `orm.Create[T]()`
- Model 嵌入 `orm.Model`，使用 `gai:"..."` 标签
- 中间件必须调用 `c.Next()`
- 不在 `database/driver/` 之外使用 `init()`
- 文件命名 snake_case，表名自动 plural snake_case

## 修改此框架时

- 添加新 Schema 类型 → 更新 `ai/schema/types.go` 的 GoType + NeedsImport + 3 个 driver 的 ColumnType
- 添加新 CLI 命令 → 在 `cmd/gai/` 写 cobra Command，在 `main.go` 注册
- 添加新内置中间件 → 在 `middleware/` 中返回 `ghttp.HandlerFunc`
- 添加新 Guard → 实现 `auth.Guard` 接口

## 编译与验证

```bash
go build ./... && go vet ./... && go test ./...
```
