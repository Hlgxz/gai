package lock_test

import (
	"context"
	"testing"
	"time"

	"github.com/Hlgxz/gai/lock"
)

func TestMemoryLock(t *testing.T) {
	l := lock.NewMemory()
	ctx := context.Background()
	rel, err := l.Acquire(ctx, "k", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Acquire(ctx, "k", time.Second); err != lock.ErrNotAcquired {
		t.Fatalf("want not acquired, got %v", err)
	}
	rel()
	if _, err := l.Acquire(ctx, "k", time.Second); err != nil {
		t.Fatal(err)
	}
}
