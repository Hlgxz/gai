package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/metrics"
	"github.com/Hlgxz/gai/router"
)

func TestMetricsEndpoint(t *testing.T) {
	r := router.New()
	r.Use(metrics.Middleware())
	r.Get("/hello", func(c *ghttp.Context) { c.OK(map[string]string{"ok": "1"}) })
	r.Get("/metrics", metrics.Handler())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))
	if rec.Code != 200 {
		t.Fatalf("hello: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "gai_http_requests_total") {
		t.Fatalf("metrics body: %s", body)
	}
}

func TestMountPprof(t *testing.T) {
	r := router.New()
	metrics.MountPprof(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if rec.Code == http.StatusNotFound {
		t.Fatalf("pprof not mounted: %d %s", rec.Code, rec.Body.String())
	}
}
