package metrics

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/router"
)

type counterKey struct {
	Method string
	Code   string
}

var (
	mu            sync.Mutex
	requestCounts = map[counterKey]uint64{}
	durationNanos atomic.Int64
	requestTotal  atomic.Int64
	inflight      atomic.Int64
)

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// Middleware records request counts and durations for the /metrics endpoint.
func Middleware() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/metrics") || strings.HasPrefix(c.Request.URL.Path, "/debug/pprof") {
			c.Next()
			return
		}
		start := time.Now()
		inflight.Add(1)
		sw := &statusWriter{ResponseWriter: c.Writer, code: 200}
		c.Writer = sw
		c.Next()
		inflight.Add(-1)

		elapsed := time.Since(start)
		durationNanos.Add(elapsed.Nanoseconds())
		requestTotal.Add(1)

		key := counterKey{Method: c.Request.Method, Code: fmt.Sprintf("%d", sw.code)}
		mu.Lock()
		requestCounts[key]++
		mu.Unlock()
	}
}

// Handler exposes Prometheus-style text metrics.
func Handler() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		mu.Lock()
		keys := make([]counterKey, 0, len(requestCounts))
		for k := range requestCounts {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].Method != keys[j].Method {
				return keys[i].Method < keys[j].Method
			}
			return keys[i].Code < keys[j].Code
		})
		var b strings.Builder
		b.WriteString("# HELP gai_http_requests_total Total HTTP requests\n")
		b.WriteString("# TYPE gai_http_requests_total counter\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "gai_http_requests_total{method=%q,code=%q} %d\n", k.Method, k.Code, requestCounts[k])
		}
		if len(keys) == 0 {
			b.WriteString("gai_http_requests_total{method=\"none\",code=\"0\"} 0\n")
		}
		sum := float64(durationNanos.Load()) / 1e9
		b.WriteString("# HELP gai_http_request_duration_seconds_sum Sum of request durations\n")
		b.WriteString("# TYPE gai_http_request_duration_seconds_sum counter\n")
		fmt.Fprintf(&b, "gai_http_request_duration_seconds_sum %g\n", sum)
		b.WriteString("# HELP gai_http_requests_in_flight In-flight HTTP requests\n")
		b.WriteString("# TYPE gai_http_requests_in_flight gauge\n")
		fmt.Fprintf(&b, "gai_http_requests_in_flight %d\n", inflight.Load())
		mu.Unlock()
		c.SetHeader("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, "%s", b.String())
	}
}

// MountPprof registers net/http/pprof under /debug/pprof.
func MountPprof(r *router.Router) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
	}
	wrap := func(c *ghttp.Context) {
		mux.ServeHTTP(c.Writer, c.Request)
	}
	r.Any("/debug/pprof", wrap)
	r.Any("/debug/pprof/*", wrap)
}
