package queue_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hlgxz/gai/queue"
)

func TestRetryThenFail(t *testing.T) {
	mem := queue.NewMemory(8)
	mgr := queue.New(mem).SetMaxTries(3).SetTimeout(time.Second)
	var n atomic.Int32
	mgr.Register("boom", func(ctx context.Context, payload []byte) error {
		n.Add(1)
		return errors.New("nope")
	})
	if err := mgr.Dispatch("boom", []byte("x")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	go func() { _ = mgr.Work(ctx) }()

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if n.Load() >= 3 && len(mgr.Failed()) == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("attempts=%d failed=%d", n.Load(), len(mgr.Failed()))
}
