package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

// statusWriter wraps http.ResponseWriter to capture the written status code
// while preserving optional interfaces (Flusher, Hijacker).
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("gai: underlying ResponseWriter does not implement http.Hijacker")
}

// Logger returns middleware that logs each request with method, path,
// status code, and duration using the standard slog package.
func Logger() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		sw := &statusWriter{ResponseWriter: c.Writer, code: 200}
		c.Writer = sw

		c.Next()

		slog.Info("request",
			"method", method,
			"path", path,
			"status", sw.code,
			"duration", time.Since(start).String(),
			"ip", c.ClientIP(),
		)
	}
}
