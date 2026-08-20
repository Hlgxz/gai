package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config describes a slog channel (stdout, stderr, or rotating file).
type Config struct {
	Level      string // debug, info, warn, error
	Output     string // stdout, stderr, file
	Path       string // used when Output is "file"
	MaxSize    int    // megabytes; 0 = 100
	MaxBackups int    // 0 = 3
	MaxAge     int    // days; 0 = 28
	Compress   bool
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
	if v, ok := c.Get("trace_id"); ok {
		attrs = append(attrs, "trace_id", v)
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
		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 3
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 28
		}
		return &lumberjack.Logger{
			Filename:   path,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   cfg.Compress,
		}
	default:
		return os.Stdout
	}
}
