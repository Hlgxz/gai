package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
)

// Config describes a slog channel (stdout, stderr, or file).
type Config struct {
	Level  string // debug, info, warn, error
	Output string // stdout, stderr, file
	Path   string // used when Output is "file"
}

// Setup installs a JSON slog default logger and returns it.
func Setup(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)
	w := writer(cfg)
	logger := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return logger
}

// FromRequest returns a logger annotated with the request ID when present.
func FromRequest(c *ghttp.Context) *slog.Logger {
	attrs := []any{}
	if v, ok := c.Get("request_id"); ok {
		attrs = append(attrs, "request_id", v)
	}
	if c.Request != nil {
		attrs = append(attrs, "method", c.Request.Method, "path", c.Request.URL.Path)
	}
	return slog.Default().With(attrs...)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func writer(cfg Config) io.Writer {
	switch strings.ToLower(cfg.Output) {
	case "stderr":
		return os.Stderr
	case "file":
		path := cfg.Path
		if path == "" {
			path = "storage/logs/app.log"
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return os.Stdout
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return os.Stdout
		}
		return f
	default:
		return os.Stdout
	}
}
