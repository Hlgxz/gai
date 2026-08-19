package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	ghttp "github.com/Hlgxz/gai/http"
)

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.gz.Write(b)
}

func (w *gzipWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipWriter) Flush() {
	_ = w.gz.Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Gzip compresses responses when the client accepts gzip.
func Gzip() ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		if !strings.Contains(c.Header("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)
		defer func() {
			_ = gz.Close()
			gz.Reset(io.Discard)
			gzipPool.Put(gz)
		}()

		c.SetHeader("Content-Encoding", "gzip")
		c.SetHeader("Vary", "Accept-Encoding")
		c.Writer = &gzipWriter{ResponseWriter: c.Writer, gz: gz}
		c.Next()
	}
}
