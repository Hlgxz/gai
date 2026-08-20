package middleware

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// SecurityConfig controls standard browser security headers.
type SecurityConfig struct {
	ContentTypeNosniff bool
	FrameOptions       string // DENY, SAMEORIGIN, or empty to skip
	XSSProtection      string
	ReferrerPolicy     string
	HSTS               string // e.g. "max-age=31536000; includeSubDomains"
	CSP                string
}

// DefaultSecurityConfig is a conservative API-oriented set of headers.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentTypeNosniff: true,
		FrameOptions:       "DENY",
		XSSProtection:      "0",
		ReferrerPolicy:     "no-referrer",
	}
}

// SecurityHeaders sets X-Content-Type-Options, X-Frame-Options, and related headers.
func SecurityHeaders(cfgs ...SecurityConfig) ghttp.HandlerFunc {
	cfg := DefaultSecurityConfig()
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return func(c *ghttp.Context) {
		h := c.Writer.Header()
		if cfg.ContentTypeNosniff {
			h.Set("X-Content-Type-Options", "nosniff")
		}
		if cfg.FrameOptions != "" {
			h.Set("X-Frame-Options", cfg.FrameOptions)
		}
		if cfg.XSSProtection != "" {
			h.Set("X-XSS-Protection", cfg.XSSProtection)
		}
		if cfg.ReferrerPolicy != "" {
			h.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}
		if cfg.HSTS != "" {
			h.Set("Strict-Transport-Security", cfg.HSTS)
		}
		if cfg.CSP != "" {
			h.Set("Content-Security-Policy", cfg.CSP)
		}
		c.Next()
	}
}
