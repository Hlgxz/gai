package schedule

import (
	"context"
	"sync"
	"time"
)

type task struct {
	name     string
	interval time.Duration
	dailyAt  string // "15:04"
	fn       func(context.Context)
}

// Scheduler runs interval and daily jobs in-process.
type Scheduler struct {
	mu    sync.Mutex
	tasks []task
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Every(d time.Duration, fn func(context.Context)) *Scheduler {
	s.mu.Lock()
	s.tasks = append(s.tasks, task{interval: d, fn: fn})
	s.mu.Unlock()
	return s
}

func (s *Scheduler) Hourly(fn func(context.Context)) *Scheduler {
	return s.Every(time.Hour, fn)
}

func (s *Scheduler) DailyAt(hhmm string, fn func(context.Context)) *Scheduler {
	s.mu.Lock()
	s.tasks = append(s.tasks, task{dailyAt: hhmm, fn: fn})
	s.mu.Unlock()
	return s
}

// Run blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()
	tasks := append([]task(nil), s.tasks...)
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, t := range tasks {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			if t.dailyAt != "" {
				runDaily(ctx, t.dailyAt, t.fn)
				return
			}
			runEvery(ctx, t.interval, t.fn)
		}()
	}
	wg.Wait()
}

func runEvery(ctx context.Context, d time.Duration, fn func(context.Context)) {
	if d <= 0 {
		d = time.Minute
	}
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(ctx)
		}
	}
}

func runDaily(ctx context.Context, hhmm string, fn func(context.Context)) {
	for {
		wait := timeUntil(hhmm)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			fn(ctx)
		}
	}
}

func timeUntil(hhmm string) time.Duration {
	now := time.Now()
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Hour
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
