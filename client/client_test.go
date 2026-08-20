package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hlgxz/gai/client"
)

func TestGetJSON(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.New()
	c.Retries = 2
	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("got %#v", out)
	}
	if n != 2 {
		t.Fatalf("retries: %d", n)
	}
}
