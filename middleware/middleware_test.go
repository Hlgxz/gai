package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/middleware"
	"github.com/Hlgxz/gai/router"
)

func TestRecovery(t *testing.T) {
	r := router.New()
	r.Use(middleware.Recovery())
	r.Get("/panic", func(c *ghttp.Context) {
		panic("boom")
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestRequestID(t *testing.T) {
	r := router.New()
	r.Use(middleware.RequestID())
	r.Get("/id", func(c *ghttp.Context) {
		c.String(200, "%s", middleware.RequestIDFrom(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	req.Header.Set(middleware.RequestIDHeader, "fixed-id")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Header().Get(middleware.RequestIDHeader) != "fixed-id" {
		t.Fatalf("header: %s", rec.Header().Get(middleware.RequestIDHeader))
	}
	if rec.Body.String() != "fixed-id" {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	r := router.New()
	r.Use(middleware.CORS())
	r.Get("/ok", func(c *ghttp.Context) { c.OK(map[string]string{"ok": "1"}) })
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("missing CORS header: %+v", rec.Header())
	}
}

func TestSecurityHeaders(t *testing.T) {
	r := router.New()
	r.Use(middleware.SecurityHeaders())
	r.Get("/x", func(c *ghttp.Context) { c.String(200, "ok") })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers: %v", rec.Header())
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("frame: %s", rec.Header().Get("X-Frame-Options"))
	}
}
