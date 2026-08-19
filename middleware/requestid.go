package middleware

import (
	"crypto/rand"
	"encoding/hex"

	ghttp "github.com/Hlgxz/gai/http"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDKey = "request_id"

// RequestID assigns a unique ID to each request (reuses inbound header if present).
func RequestID() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		id := c.Header(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.SetHeader(RequestIDHeader, id)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "gai-req"
	}
	return hex.EncodeToString(b)
}

// RequestIDFrom returns the request ID stored on the context, or empty string.
func RequestIDFrom(c *ghttp.Context) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
