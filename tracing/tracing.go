package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

type ctxKey struct{}

// Span is a lightweight OpenTelemetry-style span.
type Span interface {
	End()
	SetAttribute(key string, value any)
	RecordError(err error)
	TraceID() string
	SpanID() string
	Context() context.Context
}

type span struct {
	ctx      context.Context
	name     string
	traceID  string
	spanID   string
	parentID string
	start    time.Time
	attrs    map[string]any
	err      error
	ended    bool
	onEnd    func(*span)
}

func (s *span) End() {
	if s.ended {
		return
	}
	s.ended = true
	if s.onEnd != nil {
		s.onEnd(s)
	} else {
		args := []any{
			"span", s.name,
			"trace_id", s.traceID,
			"span_id", s.spanID,
			"duration_ms", time.Since(s.start).Milliseconds(),
		}
		if s.parentID != "" {
			args = append(args, "parent_id", s.parentID)
		}
		for k, v := range s.attrs {
			args = append(args, k, v)
		}
		if s.err != nil {
			slog.Error("span", append(args, "error", s.err.Error())...)
			return
		}
		slog.Debug("span", args...)
	}
}

func (s *span) SetAttribute(key string, value any) {
	if s.attrs == nil {
		s.attrs = map[string]any{}
	}
	s.attrs[key] = value
}

func (s *span) RecordError(err error) { s.err = err }

func (s *span) TraceID() string { return s.traceID }

func (s *span) SpanID() string { return s.spanID }

func (s *span) Context() context.Context { return s.ctx }

// Config controls tracing setup.
type Config struct {
	ServiceName string
	Enabled     bool
}

var defaultOnEnd func(*span)

// Setup installs the process-wide tracer. Endpoint-based OTLP export can be
// layered by replacing the span exporter via SetExporter.
func Setup(cfg Config) {
	if !cfg.Enabled {
		defaultOnEnd = nil
		return
	}
	svc := cfg.ServiceName
	if svc == "" {
		svc = "gai"
	}
	defaultOnEnd = func(s *span) {
		args := []any{
			"service", svc,
			"span", s.name,
			"trace_id", s.traceID,
			"span_id", s.spanID,
			"duration_ms", time.Since(s.start).Milliseconds(),
		}
		if s.parentID != "" {
			args = append(args, "parent_span_id", s.parentID)
		}
		for k, v := range s.attrs {
			args = append(args, k, v)
		}
		if s.err != nil {
			slog.Info("otel.span", append(args, "status", "error", "error", s.err.Error())...)
			return
		}
		slog.Info("otel.span", append(args, "status", "ok")...)
	}
}

// SetExporter replaces the span-end callback (for OTLP adapters).
func SetExporter(fn func(name, traceID, spanID, parentID string, duration time.Duration, attrs map[string]any, err error)) {
	if fn == nil {
		defaultOnEnd = nil
		return
	}
	defaultOnEnd = func(s *span) {
		fn(s.name, s.traceID, s.spanID, s.parentID, time.Since(s.start), s.attrs, s.err)
	}
}

// Start creates a child span. Trace context is taken from ctx or generated.
func Start(ctx context.Context, name string) (context.Context, Span) {
	parent, _ := SpanFromContext(ctx)
	s := &span{
		name:  name,
		start: time.Now(),
		onEnd: defaultOnEnd,
	}
	if parent != nil {
		s.traceID = parent.TraceID()
		s.parentID = parent.SpanID()
	} else if tid, sid, ok := extractTraceparent(ctx); ok {
		s.traceID = tid
		s.parentID = sid
	} else {
		s.traceID = newID(16)
	}
	s.spanID = newID(8)
	s.ctx = context.WithValue(ctx, ctxKey{}, s)
	return s.ctx, s
}

// SpanFromContext returns the active span, if any.
func SpanFromContext(ctx context.Context) (Span, bool) {
	if ctx == nil {
		return nil, false
	}
	s, ok := ctx.Value(ctxKey{}).(*span)
	if !ok || s == nil {
		return nil, false
	}
	return s, true
}

// Middleware starts a SERVER span per request and injects W3C traceparent.
func Middleware() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		ctx := c.Request.Context()
		if tp := c.Header("traceparent"); tp != "" {
			ctx = withTraceparent(ctx, tp)
		}
		ctx, span := Start(ctx, c.Request.Method+" "+c.Request.URL.Path)
		span.SetAttribute("http.method", c.Request.Method)
		span.SetAttribute("http.route", c.Request.URL.Path)
		c.Request = c.Request.WithContext(ctx)
		c.Set("trace_id", span.TraceID())
		c.SetHeader("traceparent", formatTraceparent(span.TraceID(), span.SpanID()))
		defer func() {
			if rec := recover(); rec != nil {
				span.RecordError(fmt.Errorf("%v", rec))
				span.End()
				panic(rec)
			}
			span.SetAttribute("http.status_code", c.StatusCode())
			span.End()
		}()
		c.Next()
	}
}

func newID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func formatTraceparent(traceID, spanID string) string {
	return "00-" + traceID + "-" + spanID + "-01"
}

type tpKey struct{}

func withTraceparent(ctx context.Context, header string) context.Context {
	return context.WithValue(ctx, tpKey{}, header)
}

func extractTraceparent(ctx context.Context) (traceID, spanID string, ok bool) {
	v, _ := ctx.Value(tpKey{}).(string)
	parts := strings.Split(v, "-")
	if len(parts) < 4 || len(parts[1]) != 32 || len(parts[2]) != 16 {
		return "", "", false
	}
	return parts[1], parts[2], true
}
