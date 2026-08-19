package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/Hlgxz/gai/cache"
)

func TestMemoryCache(t *testing.T) {
	m := &cache.Manager{Store: cache.NewMemory()}
	ctx := context.Background()
	if err := m.SetString(ctx, "k", "v", time.Second); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetString(ctx, "k")
	if err != nil || got != "v" {
		t.Fatalf("got %q %v", got, err)
	}
	ok, _ := m.Has(ctx, "k")
	if !ok {
		t.Fatal("has")
	}
	_ = m.Delete(ctx, "k")
	if _, err := m.Get(ctx, "k"); err != cache.ErrMiss {
		t.Fatalf("want miss got %v", err)
	}
}
