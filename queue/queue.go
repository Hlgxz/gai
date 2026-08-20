package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Job is a unit of work identified by name with an opaque payload.
type Job struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Payload  []byte        `json:"payload"`
	Delay    time.Duration `json:"-"`
	Attempts int           `json:"attempts"`
	MaxTries int           `json:"max_tries"`
	Timeout  time.Duration `json:"timeout_ns"`
	Error    string        `json:"error,omitempty"`
}

// Handler processes a job.
type Handler func(ctx context.Context, payload []byte) error

// Queue is a job backend.
type Queue interface {
	Push(job Job) error
	Pop(ctx context.Context) (*Job, error)
	Fail(job Job, cause error) error
	Retry(job Job, delay time.Duration) error
}

// FailedSink is implemented by backends that can list failed jobs.
type FailedSink interface {
	Failed() []Job
}

// Manager registers handlers and dispatches jobs.
type Manager struct {
	q           Queue
	mu          sync.RWMutex
	handlers    map[string]Handler
	maxTries    int
	timeout     time.Duration
	concurrency int
	backoff     func(attempts int) time.Duration
	OnFailed    func(job Job, err error)
}

func New(q Queue) *Manager {
	return &Manager{
		q:           q,
		handlers:    make(map[string]Handler),
		maxTries:    3,
		timeout:     60 * time.Second,
		concurrency: 1,
		backoff: func(attempts int) time.Duration {
			d := time.Second * time.Duration(1<<uint(attempts-1))
			if d > 30*time.Second {
				d = 30 * time.Second
			}
			return d
		},
	}
}

func (m *Manager) SetMaxTries(n int) *Manager {
	if n > 0 {
		m.maxTries = n
	}
	return m
}

func (m *Manager) SetTimeout(d time.Duration) *Manager {
	m.timeout = d
	return m
}

func (m *Manager) SetConcurrency(n int) *Manager {
	if n > 0 {
		m.concurrency = n
	}
	return m
}

func (m *Manager) Register(name string, h Handler) {
	m.mu.Lock()
	m.handlers[name] = h
	m.mu.Unlock()
}

func (m *Manager) Dispatch(name string, payload []byte) error {
	return m.q.Push(Job{ID: newJobID(), Name: name, Payload: payload, MaxTries: m.maxTries, Timeout: m.timeout})
}

func (m *Manager) Later(d time.Duration, name string, payload []byte) error {
	return m.q.Push(Job{ID: newJobID(), Name: name, Payload: payload, Delay: d, MaxTries: m.maxTries, Timeout: m.timeout})
}

// Failed returns jobs that exhausted retries, when the backend supports it.
func (m *Manager) Failed() []Job {
	if s, ok := m.q.(FailedSink); ok {
		return s.Failed()
	}
	return nil
}

// Work pops jobs until ctx is cancelled.
func (m *Manager) Work(ctx context.Context) error {
	n := m.concurrency
	if n < 1 {
		n = 1
	}
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		job, err := m.q.Pop(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				continue
			}
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(j *Job) {
			defer wg.Done()
			defer func() { <-sem }()
			m.process(ctx, j)
		}(job)
	}
}

func (m *Manager) process(parent context.Context, job *Job) {
	m.mu.RLock()
	h := m.handlers[job.Name]
	m.mu.RUnlock()
	if h == nil {
		slog.Warn("gai/queue: no handler", "job", job.Name, "id", job.ID)
		return
	}

	job.Attempts++
	timeout := job.Timeout
	if timeout <= 0 {
		timeout = m.timeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	err := h(ctx, job.Payload)
	if err == nil {
		return
	}

	max := job.MaxTries
	if max <= 0 {
		max = m.maxTries
	}
	if job.Attempts < max {
		delay := m.backoff(job.Attempts)
		if rerr := m.q.Retry(*job, delay); rerr != nil {
			slog.Error("gai/queue: retry failed", "job", job.Name, "error", rerr)
		}
		return
	}

	job.Error = err.Error()
	if ferr := m.q.Fail(*job, err); ferr != nil {
		slog.Error("gai/queue: fail persist", "job", job.Name, "error", ferr)
	}
	if m.OnFailed != nil {
		m.OnFailed(*job, err)
	}
	slog.Error("gai/queue: job failed", "job", job.Name, "id", job.ID, "attempts", job.Attempts, "error", err)
}

func newJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type delayed struct {
	job Job
	at  time.Time
}

// Memory is an in-process queue (useful in tests and single-node apps).
type Memory struct {
	ch     chan delayed
	mu     sync.Mutex
	failed []Job
}

func NewMemory(size int) *Memory {
	if size <= 0 {
		size = 64
	}
	return &Memory{ch: make(chan delayed, size)}
}

func (m *Memory) Push(job Job) error {
	at := time.Now().Add(job.Delay)
	select {
	case m.ch <- delayed{job: job, at: at}:
		return nil
	default:
		return errQueueFull
	}
}

func (m *Memory) Pop(ctx context.Context) (*Job, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case d := <-m.ch:
			wait := time.Until(d.at)
			if wait > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(wait):
				}
			}
			j := d.job
			return &j, nil
		}
	}
}

func (m *Memory) Fail(job Job, cause error) error {
	if cause != nil && job.Error == "" {
		job.Error = cause.Error()
	}
	m.mu.Lock()
	m.failed = append(m.failed, job)
	m.mu.Unlock()
	return nil
}

func (m *Memory) Retry(job Job, delay time.Duration) error {
	job.Delay = delay
	return m.Push(job)
}

func (m *Memory) Failed() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Job, len(m.failed))
	copy(out, m.failed)
	return out
}

var errQueueFull = errString("gai/queue: full")

type errString string

func (e errString) Error() string { return string(e) }

func encodeJob(j Job) ([]byte, error) {
	return json.Marshal(j)
}

func decodeJob(b []byte) (Job, error) {
	var j Job
	if err := json.Unmarshal(b, &j); err != nil {
		return Job{}, fmt.Errorf("gai/queue: decode: %w", err)
	}
	return j, nil
}
