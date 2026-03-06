package middleware

import (
	"log/slog"
	"net/http"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

// statusWriter wraps http.ResponseWriter to capture the written status code.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
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
