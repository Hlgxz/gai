package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectModule(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("go.mod", []byte("module example.com/demo\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectModule(); got != "example.com/demo" {
		t.Fatalf("got %q", got)
	}
}

func TestCreateProjectVersions(t *testing.T) {
	parent := t.TempDir()
	t.Chdir(parent)
	if err := createProject("demo", "example.com/demo", 9090); err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join("demo", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(mod)
	if !strings.Contains(text, "go 1.24") {
		t.Fatalf("go version: %s", text)
	}
	if !strings.Contains(text, "github.com/Hlgxz/gai v"+version) {
		t.Fatalf("gai version: %s", text)
	}
	gen := filepath.Join("demo", "routes", "generated.go")
	if _, err := os.Stat(gen); err != nil {
		t.Fatal(err)
	}
	yaml, err := os.ReadFile(filepath.Join("demo", "config", "app.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yaml), "log:") {
		t.Fatalf("missing log config: %s", yaml)
	}
	if !strings.Contains(string(yaml), "cache:") {
		t.Fatalf("missing cache config: %s", yaml)
	}
	mainSrc, err := os.ReadFile(filepath.Join("demo", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSrc), "UseServices") {
		t.Fatalf("main should call UseServices:\n%s", mainSrc)
	}
}

func TestFormatGoSource(t *testing.T) {
	out := formatGoSource("package main\nfunc main(){ }\n")
	if !strings.Contains(out, "package main") {
		t.Fatalf("formatted: %s", out)
	}
}
