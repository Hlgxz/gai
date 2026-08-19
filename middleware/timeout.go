package middleware

import (
	"context"
	"net/http"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

// Timeout sets a deadline on the request context. Handlers should honour c.Ctx().
func Timeout(d time.Duration) ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() == context.DeadlineExceeded && !c.IsWritten() {
			c.JSON(http.StatusGatewayTimeout, map[string]any{
				"code":    504,
				"message": "Request Timeout",
			})
		}
	}
}
