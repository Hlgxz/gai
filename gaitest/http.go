// Package gaitest provides HTTP test helpers and fakes for Gai applications.
package gaitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Hlgxz/gai/mail"
	"github.com/Hlgxz/gai/queue"
)

// Response is the captured result of a test request.
type Response struct {
	Code   int
	Header http.Header
	Body   []byte
}

// Perform sends a request through h and returns the recorded response.
func Perform(h http.Handler, method, path string, body any) *Response {
	var rdr io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case string:
		rdr = strings.NewReader(b)
		contentType = "text/plain"
	case []byte:
		rdr = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			panic(err)
		}
		rdr = bytes.NewReader(raw)
		contentType = "application/json"
	}

	req := httptest.NewRequest(method, path, rdr)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return &Response{
		Code:   rec.Code,
		Header: rec.Header().Clone(),
		Body:   rec.Body.Bytes(),
	}
}

// JSONMap decodes the body as a generic JSON object.
func (r *Response) JSONMap() map[string]any {
	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		return nil
	}
	return out
}

// AssertStatus fails t if the status code does not match.
func (r *Response) AssertStatus(t testing.TB, code int) {
	t.Helper()
	if r.Code != code {
		t.Fatalf("status: got %d want %d body=%s", r.Code, code, r.Body)
	}
}

// AssertOK fails t unless the status is 200.
func (r *Response) AssertOK(t testing.TB) {
	t.Helper()
	r.AssertStatus(t, http.StatusOK)
}

// AssertJSONPath fails t unless the dotted key equals want (string/number/bool).
func (r *Response) AssertJSONPath(t testing.TB, path string, want any) {
	t.Helper()
	cur := any(r.JSONMap())
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("json path %q: not an object at %s", path, part)
		}
		cur, ok = m[part]
		if !ok {
			t.Fatalf("json path %q: missing key %s in %s", path, part, r.Body)
		}
	}
	if fmtVal(cur) != fmtVal(want) {
		t.Fatalf("json path %q: got %#v want %#v", path, cur, want)
	}
}

func fmtVal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// NewMailRecorder returns an in-memory mailer for tests.
func NewMailRecorder() *mail.LogMailer {
	return &mail.LogMailer{}
}

// NewQueue returns an in-memory queue for tests.
func NewQueue() *queue.Memory {
	return queue.NewMemory(32)
}
