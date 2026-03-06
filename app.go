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
	"github.com/Hlgxz/gai/middleware"
	"github.com/Hlgxz/gai/router"
)

// Application is the heart of the Gai framework, combining the DI container,
// configuration, router, and lifecycle management into a single entry point.
type Application struct {
	*Container

	config    *config.Manager
	router    *router.Router
	providers []ServiceProvider
	booted    bool
	basePath  string
}

// New creates a new Gai application instance.
func New() *Application {
	app := &Application{
		Container: newContainer(),
		config:    config.New(),
		router:    router.New(),
	}

	// Self-register the app so providers can resolve it.
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
	dir, _ := os.Getwd()
	return dir
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
	_ = config.LoadEnvFile(app.BasePath() + "/.env")
	if err := app.config.Load(dir); err != nil {
		slog.Warn("failed to load config", "dir", dir, "error", err)
	}
	return app
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

// UseDefaults wires up the standard middleware stack (Recovery, Logger, CORS).
func (app *Application) UseDefaults() *Application {
	app.router.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS(),
	)
	return app
}

// Serve starts the HTTP server and blocks until a shutdown signal is received.
// It performs graceful shutdown with a 10-second timeout.
func (app *Application) Serve(addr string) error {
	app.Boot()

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("gai server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("gai: shutdown error: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
