package middleware

import (
	"net/http"
	"strconv"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimit is a multi-instance token counter using INCR + EXPIRE.
func RedisRateLimit(client redis.Cmdable, limit int, window time.Duration) ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		key := "gai:ratelimit:" + c.ClientIP()
		ctx := c.Ctx()
		n, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if n == 1 {
			_ = client.Expire(ctx, key, window).Err()
		}
		if n > int64(limit) {
			c.SetHeader("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, map[string]any{
				"code":    429,
				"message": "Too Many Requests",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
