package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/router"
)

func TestRouteParamAndNamed(t *testing.T) {
	r := router.New()
	r.Get("/users/:id", func(c *ghttp.Context) {
		c.String(200, "id=%s", c.Param("id"))
	}).As("users.show")

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "id=42" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if got := r.URL("users.show", map[string]string{"id": "9"}); got != "/users/9" {
		t.Fatalf("url=%s", got)
	}
}

func TestMethodNotAllowedAndHEAD(t *testing.T) {
	r := router.New()
	r.Get("/ping", func(c *ghttp.Context) {
		c.String(200, "pong")
	})

	req := httptest.NewRequest(http.MethodPost, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodHead, "/ping", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("head status %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if len(body) != 0 {
		t.Fatalf("head should have empty body, got %q", body)
	}
}

func TestTrustedProxy(t *testing.T) {
	r := router.New()
	_ = r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Get("/ip", func(c *ghttp.Context) {
		c.String(200, "%s", c.ClientIP())
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Body.String() != "10.0.0.1" {
		t.Fatalf("untrusted proxy leaked XFF: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "127.0.0.1:9"
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Body.String() != "1.1.1.1" {
		t.Fatalf("trusted proxy ignored XFF: %s", rec.Body.String())
	}
}
