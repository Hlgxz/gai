package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hlgxz/gai/lock"
)

type task struct {
	name      string
	interval  time.Duration
	dailyAt   string // "15:04"
	cron      *cronExpr
	fn        func(context.Context)
	noOverlap bool
	running   atomic.Bool
}

// Scheduler runs interval, daily, and cron jobs in-process.
type Scheduler struct {
	mu    sync.Mutex
	tasks []task
	lock  lock.Locker
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) UseLock(l lock.Locker) *Scheduler {
	s.lock = l
	return s
}

func (s *Scheduler) Every(d time.Duration, fn func(context.Context)) *Scheduler {
	s.mu.Lock()
	s.tasks = append(s.tasks, task{name: fmt.Sprintf("every-%d", len(s.tasks)), interval: d, fn: fn})
	s.mu.Unlock()
	return s
}

func (s *Scheduler) Hourly(fn func(context.Context)) *Scheduler {
	return s.Every(time.Hour, fn)
}

func (s *Scheduler) DailyAt(hhmm string, fn func(context.Context)) *Scheduler {
	s.mu.Lock()
	s.tasks = append(s.tasks, task{name: "daily-" + hhmm, dailyAt: hhmm, fn: fn})
	s.mu.Unlock()
	return s
}

// Cron registers a 5-field cron expression (min hour day month weekday).
func (s *Scheduler) Cron(expr string, fn func(context.Context)) *Scheduler {
	parsed, err := parseCron(expr)
	if err != nil {
		slog.Error("gai/schedule: invalid cron", "expr", expr, "error", err)
		return s
	}
	s.mu.Lock()
	s.tasks = append(s.tasks, task{name: "cron-" + expr, cron: parsed, fn: fn})
	s.mu.Unlock()
	return s
}

// Name sets the name of the last registered task (used for distributed locks).
func (s *Scheduler) Name(name string) *Scheduler {
	s.mu.Lock()
	if n := len(s.tasks); n > 0 {
		s.tasks[n-1].name = name
	}
	s.mu.Unlock()
	return s
}

// WithoutOverlapping skips a tick when the previous run is still in progress.
func (s *Scheduler) WithoutOverlapping() *Scheduler {
	s.mu.Lock()
	if n := len(s.tasks); n > 0 {
		s.tasks[n-1].noOverlap = true
	}
	s.mu.Unlock()
	return s
}

// Run blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()
	tasks := append([]task(nil), s.tasks...)
	locker := s.lock
	s.mu.Unlock()

	var wg sync.WaitGroup
	for i := range tasks {
		t := &tasks[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runTask(ctx, t, locker)
		}()
	}
	wg.Wait()
}

func (s *Scheduler) runTask(ctx context.Context, t *task, locker lock.Locker) {
	if t.cron != nil {
		runCron(ctx, t, locker)
		return
	}
	if t.dailyAt != "" {
		runDaily(ctx, t, locker)
		return
	}
	runEvery(ctx, t, locker)
}

func invoke(ctx context.Context, t *task, locker lock.Locker) {
	if t.noOverlap && !t.running.CompareAndSwap(false, true) {
		return
	}
	if t.noOverlap {
		defer t.running.Store(false)
	}
	if locker != nil && t.name != "" {
		rel, err := locker.Acquire(ctx, "schedule:"+t.name, time.Minute)
		if err != nil {
			return
		}
		defer rel()
	}
	t.fn(ctx)
}

func runEvery(ctx context.Context, t *task, locker lock.Locker) {
	d := t.interval
	if d <= 0 {
		d = time.Minute
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			invoke(ctx, t, locker)
		}
	}
}

func runDaily(ctx context.Context, t *task, locker lock.Locker) {
	for {
		wait := timeUntil(t.dailyAt)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			invoke(ctx, t, locker)
		}
	}
}

func runCron(ctx context.Context, t *task, locker lock.Locker) {
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if t.cron.matches(time.Now()) {
				invoke(ctx, t, locker)
			}
		}
	}
}

func timeUntil(hhmm string) time.Duration {
	now := time.Now()
	parsed, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Hour
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

type cronField struct {
	all   bool
	items map[int]struct{}
}

type cronExpr struct {
	min, hour, day, month, week cronField
}

func parseCron(expr string) (*cronExpr, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("gai/schedule: cron must have 5 fields")
	}
	c := &cronExpr{}
	var err error
	if c.min, err = parseField(parts[0], 0, 59); err != nil {
		return nil, err
	}
	if c.hour, err = parseField(parts[1], 0, 23); err != nil {
		return nil, err
	}
	if c.day, err = parseField(parts[2], 1, 31); err != nil {
		return nil, err
	}
	if c.month, err = parseField(parts[3], 1, 12); err != nil {
		return nil, err
	}
	if c.week, err = parseField(parts[4], 0, 6); err != nil {
		return nil, err
	}
	return c, nil
}

func parseField(s string, min, max int) (cronField, error) {
	if s == "*" {
		return cronField{all: true}, nil
	}
	f := cronField{items: make(map[int]struct{})}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		step := 1
		rangePart := part
		if i := strings.Index(part, "/"); i >= 0 {
			rangePart = part[:i]
			n, err := strconv.Atoi(part[i+1:])
			if err != nil || n <= 0 {
				return f, fmt.Errorf("gai/schedule: bad step %q", part)
			}
			step = n
			if rangePart == "*" {
				for v := min; v <= max; v += step {
					f.items[v] = struct{}{}
				}
				continue
			}
		}
		if strings.Contains(rangePart, "-") {
			bounds := strings.SplitN(rangePart, "-", 2)
			a, err1 := strconv.Atoi(bounds[0])
			b, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil {
				return f, fmt.Errorf("gai/schedule: bad range %q", part)
			}
			for v := a; v <= b; v += step {
				if v >= min && v <= max {
					f.items[v] = struct{}{}
				}
			}
			continue
		}
		n, err := strconv.Atoi(rangePart)
		if err != nil {
			return f, fmt.Errorf("gai/schedule: bad value %q", part)
		}
		if n >= min && n <= max {
			f.items[n] = struct{}{}
		}
	}
	return f, nil
}

func (f cronField) match(v int) bool {
	if f.all {
		return true
	}
	_, ok := f.items[v]
	return ok
}

func (c *cronExpr) matches(t time.Time) bool {
	week := int(t.Weekday())
	return c.min.match(t.Minute()) &&
		c.hour.match(t.Hour()) &&
		c.day.match(t.Day()) &&
		c.month.match(int(t.Month())) &&
		c.week.match(week)
}
