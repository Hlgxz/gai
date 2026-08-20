package storage_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Hlgxz/gai/storage"
)

func TestLocalAndS3(t *testing.T) {
	dir := t.TempDir()
	local := &storage.Local{Root: dir, BaseURL: "/files"}
	if err := local.Put("a/b.txt", strings.NewReader("hi")); err != nil {
		t.Fatal(err)
	}
	ok, _ := local.Exists("a/b.txt")
	if !ok {
		t.Fatal("exists")
	}
	r, err := local.Get("a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r)
	_ = r.Close()
	if string(b) != "hi" {
		t.Fatalf("got %q", b)
	}
	if local.URL("a/b.txt") != "/files/a/b.txt" {
		t.Fatalf("url %s", local.URL("a/b.txt"))
	}

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(200)
		case http.MethodGet:
			_, _ = w.Write([]byte("s3"))
		case http.MethodHead:
			w.WriteHeader(200)
		case http.MethodDelete:
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()

	s3 := &storage.S3{
		Endpoint:  srv.URL,
		Region:    "us-east-1",
		Bucket:    "bkt",
		AccessKey: "AKIA",
		SecretKey: "secret",
		PathStyle: true,
		Client:    srv.Client(),
		Now:       func() time.Time { return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	if err := s3.Put("x.txt", bytes.NewReader([]byte("data"))); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotAuth, "AWS4-HMAC-SHA256") {
		t.Fatalf("auth %s", gotAuth)
	}
	gr, err := s3.Get("x.txt")
	if err != nil {
		t.Fatal(err)
	}
	gb, _ := io.ReadAll(gr)
	_ = gr.Close()
	if string(gb) != "s3" {
		t.Fatalf("get %q", gb)
	}
	ok, err = s3.Exists("x.txt")
	if err != nil || !ok {
		t.Fatalf("exists %v %v", ok, err)
	}
	if err := s3.Delete("x.txt"); err != nil {
		t.Fatal(err)
	}

	mgr := storage.NewManager()
	mgr.Add("local", local)
	d, err := mgr.Disk()
	if err != nil || d != local {
		t.Fatal(err)
	}
}
