package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

type visitor struct {
	tokens    float64
	lastVisit time.Time
}

// RateLimiter holds the state for rate limiting and exposes a Stop method
// so the background cleanup goroutine can be shut down cleanly.
type RateLimiter struct {
	cancel context.CancelFunc
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

// RateLimit returns token-bucket rate limiting middleware.
// limit is the max requests per window, and window is the refill period.
// The returned RateLimiter should be kept so its Stop method can be called
// during graceful shutdown.
func RateLimit(limit int, window time.Duration) (ghttp.HandlerFunc, *RateLimiter) {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)
	rate := float64(limit) / window.Seconds()

	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{cancel: cancel}

	go func() {
		ticker := time.NewTicker(window * 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for ip, v := range visitors {
					if time.Since(v.lastVisit) > window*2 {
						delete(visitors, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	handler := func(c *ghttp.Context) {
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

	return handler, rl
}
