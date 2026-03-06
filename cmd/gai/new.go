package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCmd() *cobra.Command {
	var module string
	var port int

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Gai project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if module == "" {
				module = "github.com/yourname/" + name
			}
			return createProject(name, module, port)
		},
	}

	cmd.Flags().StringVarP(&module, "module", "m", "", "Go module path (default: github.com/yourname/<name>)")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP server port")

	return cmd
}

func createProject(name, module string, port int) error {
	fmt.Printf("Creating new Gai project: %s\n", name)

	dirs := []string{
		"app/controllers",
		"app/models",
		"app/middleware",
		"config",
		"database/migrations",
		"routes",
		"schemas",
		"storage/logs",
		".cursor/rules",
		".github",
		".kiro",
		".gemini",
		".roo",
		".augment",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(name, dir), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"go.mod":                           goModContent(module),
		"main.go":                          mainGoContent(module, port),
		".env":                             envContent(port),
		"config/app.yaml":                  appYamlContent(name, port),
		"schemas/.gitkeep":                 "",
		"database/migrations/.gitkeep":     "",
		"routes/routes.go":                 routesGoContent(module),
		".gitignore":                       gitignoreContent(),
		".cursor/rules/gai.mdc":            cursorRulesContent(module),
		"CLAUDE.md":                        claudeContent(module),
		"AGENTS.md":                        agentsContent(module),
		".github/copilot-instructions.md":  copilotContent(module),
		".windsurfrules":                   windsurfContent(module),
		".kiro/rules.md":                   kiroContent(module),
		".gemini/style-guide.md":           geminiContent(module),
		".roo/rules.md":                    rooContent(module),
		".clinerules":                      clineContent(),
		".augment/rules.md":                augmentContent(module),
	}

	for path, content := range files {
		fullPath := filepath.Join(name, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Printf("  created: %s\n", path)
	}

	fmt.Printf("\nProject %s created successfully!\n\n", name)
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  go mod tidy")
	fmt.Println("  gai serve")
	fmt.Println()

	return nil
}

// ---------------------------------------------------------- Core Files

func goModContent(module string) string {
	return fmt.Sprintf(`module %s

go 1.22

require github.com/Hlgxz/gai v0.1.0
`, module)
}

func mainGoContent(module string, port int) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"

	"github.com/Hlgxz/gai"
	"%s/routes"
)

func main() {
	app := gai.New()
	app.LoadConfig("config")
	app.UseDefaults()

	routes.Register(app)

	addr := fmt.Sprintf(":%%d", %d)
	log.Fatal(app.Serve(addr))
}
`, module, port)
}

func envContent(port int) string {
	return fmt.Sprintf(`APP_ENV=development
APP_PORT=%d
APP_DEBUG=true

DB_DRIVER=sqlite
DB_DATABASE=storage/database.db

JWT_SECRET=change-me-to-a-random-string
JWT_TTL=7200

# WECHAT_APP_ID=
# WECHAT_APP_SECRET=
`, port)
}

func appYamlContent(name string, port int) string {
	return fmt.Sprintf(`name: %s
port: %d
env: ${APP_ENV:development}
debug: ${APP_DEBUG:true}

database:
  driver: ${DB_DRIVER:sqlite}
  dsn: ${DB_DATABASE:storage/database.db}

auth:
  default: jwt
  guards:
    jwt:
      driver: jwt
      secret: ${JWT_SECRET}
      ttl: ${JWT_TTL:7200}
`, name, port)
}

func routesGoContent(module string) string {
	_ = module
	return `package routes

import (
	"github.com/Hlgxz/gai"
	ghttp "github.com/Hlgxz/gai/http"
)

// Register sets up all application routes.
func Register(app *gai.Application) {
	r := app.Router()

	r.Get("/", func(c *ghttp.Context) {
		c.Success(map[string]string{
			"framework": "Gai",
			"version":   "0.1.0",
			"message":   "Welcome to Gai! Define once, generate everything.",
		})
	})

	r.Get("/health", func(c *ghttp.Context) {
		c.OK(map[string]string{"status": "ok"})
	})
}
`
}

func gitignoreContent() string {
	return strings.TrimSpace(`
# Binaries
*.exe
*.dll
*.so
*.dylib

# IDE
.idea/
.vscode/
*.swp
*.swo

# Environment
.env
.env.local

# Storage
storage/logs/*
storage/*.db
!storage/.gitkeep

# Build
/tmp
/dist
`) + "\n"
}

// ---------------------------------------------------------- AI Rules (generated into business projects)

func aiRulesCore(module string) string {
	return fmt.Sprintf(`## Project

- **Module**: %s
- **Framework**: Gai (github.com/Hlgxz/gai) — AI-native Go web framework
- **Docs**: https://github.com/Hlgxz/gai

## Setup

`+"```"+`bash
go mod tidy && go build ./...
`+"```"+`

## Architecture

`+"```"+`
%s/
├── app/controllers/   # HTTP 控制器
├── app/models/        # ORM 模型
├── app/middleware/     # 自定义中间件
├── config/app.yaml    # 配置 (支持 ${ENV_VAR:default})
├── database/migrations/
├── routes/routes.go   # 路由注册入口
├── schemas/           # Schema 定义 (YAML → 自动生成代码)
├── storage/           # 日志、SQLite、上传文件
├── .env               # 环境变量
└── main.go            # 入口
`+"```"+`

## Gai Framework Conventions

### 1. Import 别名 (强制)
`+"```"+`go
import ghttp "github.com/Hlgxz/gai/http"  // 必须用 ghttp 别名
`+"```"+`

### 2. Handler 签名
`+"```"+`go
func Handler(c *ghttp.Context) {
    c.Success(data)        // → {"code":0,"message":"ok","data":...}
    c.Error(400, "msg")    // → {"code":400,"message":"msg"}
}
`+"```"+`

### 3. ORM (泛型函数)
`+"```"+`go
import "github.com/Hlgxz/gai/database/orm"

// 查询
query := orm.Query[User](db).Where("status", "=", "active").OrderBy("id", "DESC")
users, _ := orm.Get[User](query)       // []User
user, _ := orm.First[User](query)      // *User
page, _ := orm.Paginate[User](query, 1, 20)

// 增删改
created, _ := orm.Create[User](db, &User{Name: "test"})
orm.Update[User](db, user)
orm.Delete[User](db, user)             // 软删除
`+"```"+`

### 4. Model 定义
`+"```"+`go
import "github.com/Hlgxz/gai/database/orm"

type User struct {
    orm.Model                                        // ID, CreatedAt, UpdatedAt, DeletedAt
    Name  string `+"`"+`json:"name"  gai:"column:name;size:100"`+"`"+`
    Email string `+"`"+`json:"email" gai:"column:email;unique"`+"`"+`
    Posts []Post `+"`"+`json:"-"     gai:"hasMany"`+"`"+`
}
`+"```"+`

### 5. 路由
`+"```"+`go
import "github.com/Hlgxz/gai/router"

r.Get("/path/:id", handler)
r.Group("/api/v1", func(g *router.Group) {
    g.Use(authManager.Middleware("jwt"))
    g.Resource("/users", userController) // 自动 CRUD 五条路由
})
`+"```"+`

### 6. 中间件
`+"```"+`go
func MyMiddleware() ghttp.HandlerFunc {
    return func(c *ghttp.Context) {
        // 前置逻辑
        c.Next()  // 必须调用
        // 后置逻辑
    }
}
`+"```"+`

### 7. 校验
`+"```"+`go
v := ghttp.NewValidator(data, map[string]string{
    "email": "required|email",
    "name":  "required|min:2|max:50",
    "phone": "phone",
})
if errs := v.Validate(); errs != nil { /* 422 */ }
`+"```"+`

### 8. Schema 驱动开发
在 schemas/ 中创建 YAML，运行 `+"`"+`gai generate --schema schemas/`+"`"+` 自动生成 Model + Controller + Migration + Routes。

## Package 速查

| 导入路径 | 别名 | 用途 |
|---------|------|------|
| github.com/Hlgxz/gai | gai | Application, Container, Make[T] |
| github.com/Hlgxz/gai/http | ghttp | Context, HandlerFunc, Validator |
| github.com/Hlgxz/gai/router | router | Router, Group, ResourceController |
| github.com/Hlgxz/gai/database/orm | orm | DB, Model, Query[T], Get[T], Create[T] |
| github.com/Hlgxz/gai/auth | auth | Manager, Guard, JWTGuard |
| github.com/Hlgxz/gai/middleware | middleware | CORS, Logger, Recovery, RateLimit |
| github.com/Hlgxz/gai/miniapp/wechat | wechat | Client, Auth, Pay, Message |
| github.com/Hlgxz/gai/miniapp/alipay | alipay | Client, Auth |
| github.com/Hlgxz/gai/support | support | Snake, Camel, Hash, Env |

## 禁止

- 导入 gai/http 不加 ghttp 别名
- 在 handler 中使用 net/http 的 ResponseWriter/Request
- 跳过中间件的 c.Next()
- 直接写 SQL（用 query builder 或 migration blueprint）
- 添加 gin/echo/chi 等第三方路由库
`, module, strings.Split(module, "/")[len(strings.Split(module, "/"))-1])
}

func cursorRulesContent(module string) string {
	return "---\ndescription: Gai framework conventions for this project. Apply to all Go files.\nglobs: \"**/*.go\"\nalwaysApply: true\n---\n\n# Gai Framework Rules\n\n" + aiRulesCore(module)
}

func claudeContent(module string) string {
	return "# CLAUDE.md — Gai Framework Project\n\n" + aiRulesCore(module)
}

func agentsContent(module string) string {
	return "# AGENTS.md — Gai Framework Project\n\n> For AI coding agents (Codex CLI, etc.)\n\n" + aiRulesCore(module)
}

func copilotContent(module string) string {
	return "# GitHub Copilot Instructions\n\n" + aiRulesCore(module)
}

func windsurfContent(module string) string {
	return "# Windsurf Rules\n\n" + aiRulesCore(module)
}

func kiroContent(module string) string {
	return "# Kiro Rules\n\n" + aiRulesCore(module)
}

func geminiContent(module string) string {
	return "# Gemini Style Guide\n\n" + aiRulesCore(module)
}

func rooContent(module string) string {
	return "# Roo Code Rules\n\n" + aiRulesCore(module)
}

func clineContent() string {
	return `# Cline / Roo Code Rules — Gai Framework
#
# Framework: github.com/Hlgxz/gai | Docs: https://github.com/Hlgxz/gai
# MUST: import github.com/Hlgxz/gai/http as ghttp
# Handler: func(c *ghttp.Context) — use c.Success(), c.Error(), c.JSON()
# ORM: orm.Query[T](db), orm.Get[T](q), orm.Create[T](db, &item)
# Model: embed orm.Model, use gai:"..." struct tags
# Route: r.Get("/:param", h), r.Group(), r.Resource()
# Middleware: return ghttp.HandlerFunc, call c.Next()
# Validate: ghttp.NewValidator(data, rules) — pipe syntax required|email|min:5
# Schema: YAML in schemas/ → gai generate --schema schemas/
# See CLAUDE.md for full reference.
`
}

func augmentContent(module string) string {
	return "# Augment / Antigravity Rules\n\n" + aiRulesCore(module)
}
