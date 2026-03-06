package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// HandlerFunc is the Gai handler signature. It receives a rich Context
// instead of the raw http.Request / http.ResponseWriter pair.
type HandlerFunc func(c *Context)

// Context wraps a single HTTP request/response cycle, providing convenient
// accessors for params, query, body, and response building.
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	// Params holds path parameters (e.g. :id).
	Params map[string]string

	// store is per-request key/value storage (like Laravel's request attributes).
	store map[string]any

	// handlers is the middleware + final handler chain.
	handlers []HandlerFunc
	index    int

	mu       sync.RWMutex
	written  bool
	status   int
}

// NewContext creates a fresh Context for the given request cycle.
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer:  w,
		Request: r,
		Params:  make(map[string]string),
		store:   make(map[string]any),
		index:   -1,
		status:  http.StatusOK,
	}
}

// ------------------------------------------------------------------ Flow

// Next advances to the next handler in the chain (used inside middleware).
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort stops the chain from continuing.
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

// AbortWithStatus stops the chain and writes a status code.
func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	c.Abort()
}

// AbortWithJSON stops the chain and writes a JSON error.
func (c *Context) AbortWithJSON(code int, obj any) {
	c.JSON(code, obj)
	c.Abort()
}

// SetHandlers is used internally by the router to inject the handler chain.
func (c *Context) SetHandlers(handlers []HandlerFunc) {
	c.handlers = handlers
}

// ---------------------------------------------------------------- Input

// Param returns a path parameter by name.
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// ParamInt returns a path parameter parsed as int, or 0 on failure.
func (c *Context) ParamInt(key string) int {
	v, _ := strconv.Atoi(c.Params[key])
	return v
}

// Query returns a query-string parameter.
func (c *Context) Query(key string, fallback ...string) string {
	val := c.Request.URL.Query().Get(key)
	if val == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return val
}

// QueryInt returns a query parameter parsed as int.
func (c *Context) QueryInt(key string, fallback ...int) int {
	val := c.Request.URL.Query().Get(key)
	if val == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return i
}

// QueryValues returns all values for a repeated query key.
func (c *Context) QueryValues(key string) []string {
	return c.Request.URL.Query()[key]
}

// PostForm returns a form field from application/x-www-form-urlencoded or
// multipart/form-data requests.
func (c *Context) PostForm(key string, fallback ...string) string {
	val := c.Request.PostFormValue(key)
	if val == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return val
}

// FormFile returns the first uploaded file for the given key.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	_, fh, err := c.Request.FormFile(key)
	return fh, err
}

// maxBodySize is the default limit for reading request bodies (10 MB).
const maxBodySize = 10 << 20

// Body reads the raw request body, limited to 10 MB by default.
func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize))
}

// BindJSON decodes the JSON request body into dst.
func (c *Context) BindJSON(dst any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// FullURL returns the full request URL.
func (c *Context) FullURL() string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.RequestURI)
}

// ClientIP extracts the client's IP address.
func (c *Context) ClientIP() string {
	if forwarded := c.Request.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.SplitN(forwarded, ",", 2)[0]
	}
	if realIP := c.Request.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return c.Request.RemoteAddr
}

// Header returns a request header value.
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// ---------------------------------------------------------------- Store

// Set stores a key-value pair on the context (visible to downstream handlers).
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	c.store[key] = value
	c.mu.Unlock()
}

// Get retrieves a value from the context store.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.store[key]
	c.mu.RUnlock()
	return v, ok
}

// MustGet retrieves a value or panics.
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic(fmt.Sprintf("gai: key %q not found in context", key))
	}
	return v
}

// Ctx returns the stdlib context.Context from the underlying request.
func (c *Context) Ctx() context.Context {
	return c.Request.Context()
}

// ------------------------------------------------------------ Response

// Status sets the HTTP status code without writing it yet.
func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) *Context {
	c.Writer.Header().Set(key, value)
	return c
}

// JSON serializes obj as JSON and writes it with the given status.
func (c *Context) JSON(code int, obj any) {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.written = true
	if err := json.NewEncoder(c.Writer).Encode(obj); err != nil {
		// Header already sent; log instead of calling http.Error which
		// would trigger a superfluous WriteHeader warning.
		slog.Error("gai: failed to encode JSON response", "error", err)
	}
}

// OK is shorthand for JSON(200, obj).
func (c *Context) OK(obj any) {
	c.JSON(http.StatusOK, obj)
}

// Success writes a standardised success envelope.
func (c *Context) Success(data any) {
	c.JSON(http.StatusOK, map[string]any{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

// Error writes a standardised error envelope.
func (c *Context) Error(code int, message string) {
	c.JSON(code, map[string]any{
		"code":    code,
		"message": message,
	})
}

// String writes a plain-text response.
func (c *Context) String(code int, format string, values ...any) {
	c.SetHeader("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.written = true
	fmt.Fprintf(c.Writer, format, values...)
}

// HTML writes an HTML response.
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.written = true
	c.Writer.Write([]byte(html))
}

// Redirect sends a redirect response.
func (c *Context) Redirect(code int, url string) {
	http.Redirect(c.Writer, c.Request, url, code)
	c.written = true
}

// NoContent sends a 204 No Content response.
func (c *Context) NoContent() {
	c.Writer.WriteHeader(http.StatusNoContent)
	c.written = true
}

// File serves a file from the filesystem.
func (c *Context) File(filepath string) {
	http.ServeFile(c.Writer, c.Request, filepath)
	c.written = true
}

// IsWritten returns whether a response has already been sent.
func (c *Context) IsWritten() bool {
	return c.written
}

// ------------------------------------------------------------ Helpers

// IsJSON returns true if the request Content-Type is JSON.
func (c *Context) IsJSON() bool {
	ct := c.Request.Header.Get("Content-Type")
	return strings.Contains(ct, "application/json")
}

// IsMethod checks the HTTP method.
func (c *Context) IsMethod(method string) bool {
	return strings.EqualFold(c.Request.Method, method)
}

// URL returns the parsed request URL.
func (c *Context) URL() *url.URL {
	return c.Request.URL
}
