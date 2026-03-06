package middleware

import (
	"net/http"
	"strconv"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
)

// CORSConfig defines the CORS middleware settings.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // seconds
}

// DefaultCORSConfig provides sane defaults allowing all origins.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		MaxAge:       86400,
	}
}

// CORS returns middleware that handles Cross-Origin Resource Sharing.
func CORS(cfgs ...CORSConfig) ghttp.HandlerFunc {
	cfg := DefaultCORSConfig()
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(c *ghttp.Context) {
		origin := c.Header("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		for _, o := range cfg.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			c.Next()
			return
		}

		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		if cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		if exposeHeaders != "" {
			h.Set("Access-Control-Expose-Headers", exposeHeaders)
		}

		if c.Request.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", allowMethods)
			h.Set("Access-Control-Allow-Headers", allowHeaders)
			h.Set("Access-Control-Max-Age", maxAge)
			c.Writer.WriteHeader(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}
