package gai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hlgxz/gai"
	ghttp "github.com/Hlgxz/gai/http"
)

type shutdownProvider struct {
	booted   bool
	shutdown bool
}

func (p *shutdownProvider) Register(app *gai.Application) {}
func (p *shutdownProvider) Boot(app *gai.Application)     { p.booted = true }
func (p *shutdownProvider) Shutdown(ctx context.Context) error {
	p.shutdown = true
	return nil
}

func TestShutdownHooksAndProvider(t *testing.T) {
	app := gai.New()
	p := &shutdownProvider{}
	app.Register(p)
	app.Boot()
	if !p.booted {
		t.Fatal("provider not booted")
	}

	called := false
	app.OnShutdown(func(ctx context.Context) error {
		called = true
		return nil
	})
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !p.shutdown {
		t.Fatal("provider shutdown not called")
	}
	if !called {
		t.Fatal("hook not called")
	}
}

func TestLivenessAndMetrics(t *testing.T) {
	app := gai.New()
	app.UseDefaults()
	app.EnableMetrics()
	app.Router().Get("/health/live", app.Liveness())
	app.Router().Get("/ping", func(c *ghttp.Context) { c.OK(map[string]string{"pong": "1"}) })

	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rec.Code != 200 {
		t.Fatalf("live: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	rec = httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != 200 {
		t.Fatalf("metrics: %d", rec.Code)
	}
}

func TestUseServices(t *testing.T) {
	app := gai.New()
	app.UseServices()
	if !app.Has("cache") || !app.Has("queue") || !app.Has("mail") || !app.Has("storage") || !app.Has("session") {
		t.Fatalf("missing bindings cache=%v queue=%v mail=%v storage=%v session=%v",
			app.Has("cache"), app.Has("queue"), app.Has("mail"), app.Has("storage"), app.Has("session"))
	}
	rec := httptest.NewRecorder()
	app.UseDefaults()
	app.Router().Get("/hdr", func(c *ghttp.Context) { c.String(200, "ok") })
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hdr", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("security headers missing: %v", rec.Header())
	}
}
