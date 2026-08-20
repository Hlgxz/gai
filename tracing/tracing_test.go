package tracing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/router"
	"github.com/Hlgxz/gai/tracing"
)

func TestStartAndMiddleware(t *testing.T) {
	tracing.Setup(tracing.Config{Enabled: true, ServiceName: "test"})
	ctx, span := tracing.Start(context.Background(), "work")
	span.SetAttribute("k", "v")
	if span.TraceID() == "" || span.SpanID() == "" {
		t.Fatal("ids")
	}
	got, ok := tracing.SpanFromContext(ctx)
	if !ok || got.TraceID() != span.TraceID() {
		t.Fatal("from context")
	}
	span.End()

	r := router.New()
	r.Use(tracing.Middleware())
	r.Get("/x", func(c *ghttp.Context) {
		c.OK(map[string]string{"ok": "1"})
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Header().Get("traceparent") == "" {
		t.Fatal("missing traceparent")
	}
}
