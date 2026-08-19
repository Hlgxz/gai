package gai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hlgxz/gai/config"
	"github.com/Hlgxz/gai/database"
	"github.com/Hlgxz/gai/database/orm"
	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/middleware"
	"github.com/Hlgxz/gai/router"
)

// Application is the heart of the Gai framework, combining the DI container,
// configuration, router, and lifecycle management into a single entry point.
type Application struct {
	*Container

	config          *config.Manager
	router          *router.Router
	providers       []ServiceProvider
	booted          bool
	basePath        string
	shutdownTimeout time.Duration
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
}

// New creates a new Gai application instance.
func New() *Application {
	app := &Application{
		Container:       newContainer(),
		config:          config.New(),
		router:          router.New(),
		shutdownTimeout: 10 * time.Second,
		readTimeout:     15 * time.Second,
		writeTimeout:    15 * time.Second,
		idleTimeout:     60 * time.Second,
	}

	app.Instance("app", app)
	app.Instance("config", app.config)
	app.Instance("router", app.router)

	return app
}

// SetBasePath sets the application root directory.
func (app *Application) SetBasePath(path string) *Application {
	app.basePath = path
	return app
}

// BasePath returns the application root directory.
func (app *Application) BasePath() string {
	if app.basePath != "" {
		return app.basePath
	}
	dir, err := os.Getwd()
	if err != nil {
		slog.Warn("failed to get working directory", "error", err)
		return "."
	}
	return dir
}

// SetShutdownTimeout configures the graceful shutdown deadline.
func (app *Application) SetShutdownTimeout(d time.Duration) *Application {
	app.shutdownTimeout = d
	return app
}

// SetTimeouts configures HTTP server read/write/idle timeouts.
func (app *Application) SetTimeouts(read, write, idle time.Duration) *Application {
	app.readTimeout = read
	app.writeTimeout = write
	app.idleTimeout = idle
	return app
}

// SetTrustedProxies forwards to the router. Use "*" only behind a known proxy.
func (app *Application) SetTrustedProxies(proxies []string) *Application {
	if err := app.router.SetTrustedProxies(proxies); err != nil {
		slog.Warn("invalid trusted proxies", "error", err)
	}
	return app
}

// Config returns the configuration manager.
func (app *Application) Config() *config.Manager {
	return app.config
}

// Router returns the HTTP router.
func (app *Application) Router() *router.Router {
	return app.router
}

// LoadConfig reads YAML config files from the given directory and loads any
// .env file from the application root.
func (app *Application) LoadConfig(dir string) *Application {
	if err := config.LoadEnvFile(app.BasePath() + "/.env"); err != nil {
		slog.Warn("failed to load .env file", "error", err)
	}
	if err := app.config.Load(dir); err != nil {
		slog.Warn("failed to load config", "dir", dir, "error", err)
	}
	return app
}

// OpenDB opens the database from app.database.* config, pings it, and binds "db".
func (app *Application) OpenDB() (*orm.DB, error) {
	cfg := database.Config{
		Driver:       app.config.GetString("app.database.driver", "sqlite"),
		DSN:          app.config.GetString("app.database.dsn", "storage/database.db"),
		MaxOpenConns: app.config.GetInt("app.database.max_open_conns", 25),
		MaxIdleConns: app.config.GetInt("app.database.max_idle_conns", 5),
	}
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	app.Instance("db", db)
	return db, nil
}

// Register adds a service provider. Registration is deferred until Boot.
func (app *Application) Register(provider ServiceProvider) *Application {
	app.providers = append(app.providers, provider)
	return app
}

// Boot initialises all registered service providers. It is called
// automatically by Serve, but can be called early if needed.
func (app *Application) Boot() *Application {
	if app.booted {
		return app
	}

	for _, p := range app.providers {
		p.Register(app)
	}
	for _, p := range app.providers {
		p.Boot(app)
	}

	app.booted = true
	return app
}

// UseDefaults wires up the standard middleware stack.
func (app *Application) UseDefaults() *Application {
	app.router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.CORS(),
	)
	return app
}

// Health returns a readiness handler that pings the bound database if present.
func (app *Application) Health() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		status := map[string]any{"status": "ok"}
		if app.Has("db") {
			db := Make[*orm.DB](app.Container, "db")
			if err := db.Ping(c.Ctx()); err != nil {
				c.JSON(http.StatusServiceUnavailable, map[string]any{
					"status":  "error",
					"message": "database unavailable",
				})
				return
			}
			status["database"] = "ok"
		}
		c.OK(status)
	}
}

// Serve starts the HTTP server and blocks until a shutdown signal is received.
func (app *Application) Serve(addr string) error {
	return app.serve(addr, "", "")
}

// ServeTLS starts an HTTPS server.
func (app *Application) ServeTLS(addr, certFile, keyFile string) error {
	return app.serve(addr, certFile, keyFile)
}

func (app *Application) serve(addr, certFile, keyFile string) error {
	app.Boot()

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.router,
		ReadTimeout:       app.readTimeout,
		WriteTimeout:      app.writeTimeout,
		IdleTimeout:       app.idleTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("gai server started", "addr", addr, "tls", certFile != "")
		var err error
		if certFile != "" {
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		return fmt.Errorf("gai: server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("gai: shutdown error: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
