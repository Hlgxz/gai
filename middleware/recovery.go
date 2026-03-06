package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	ghttp "github.com/Hlgxz/gai/http"
)

// Recovery returns middleware that recovers from panics, logs the stack trace,
// and returns a 500 Internal Server Error.
func Recovery() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"stack", string(stack),
				)

				if !c.IsWritten() {
					c.JSON(http.StatusInternalServerError, map[string]any{
						"code":    500,
						"message": "Internal Server Error",
					})
				}
			}
		}()
		c.Next()
	}
}
