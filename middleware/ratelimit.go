package middleware

import (
	"net/http"
	"sync"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

type visitor struct {
	tokens    float64
	lastVisit time.Time
}

// RateLimit returns token-bucket rate limiting middleware.
// limit is the max requests per window, and window is the refill period.
func RateLimit(limit int, window time.Duration) ghttp.HandlerFunc {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)
	rate := float64(limit) / window.Seconds()

	// Background cleanup of stale entries.
	go func() {
		for {
			time.Sleep(window * 2)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastVisit) > window*2 {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *ghttp.Context) {
		ip := c.ClientIP()

		mu.Lock()
		v, exists := visitors[ip]
		if !exists {
			v = &visitor{tokens: float64(limit), lastVisit: time.Now()}
			visitors[ip] = v
		}

		elapsed := time.Since(v.lastVisit).Seconds()
		v.tokens += elapsed * rate
		if v.tokens > float64(limit) {
			v.tokens = float64(limit)
		}
		v.lastVisit = time.Now()

		if v.tokens < 1 {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, map[string]any{
				"code":    429,
				"message": "Too Many Requests",
			})
			c.Abort()
			return
		}

		v.tokens--
		mu.Unlock()

		c.Next()
	}
}
