package queue

import (
	"context"
	"sync"
	"time"
)

// Job is a unit of work identified by name with an opaque payload.
type Job struct {
	Name    string
	Payload []byte
	Delay   time.Duration
}

// Handler processes a job.
type Handler func(ctx context.Context, payload []byte) error

// Queue is a job backend.
type Queue interface {
	Push(job Job) error
	Pop(ctx context.Context) (*Job, error)
}

// Manager registers handlers and dispatches jobs.
type Manager struct {
	q        Queue
	mu       sync.RWMutex
	handlers map[string]Handler
}

func New(q Queue) *Manager {
	return &Manager{q: q, handlers: make(map[string]Handler)}
}

func (m *Manager) Register(name string, h Handler) {
	m.mu.Lock()
	m.handlers[name] = h
	m.mu.Unlock()
}

func (m *Manager) Dispatch(name string, payload []byte) error {
	return m.q.Push(Job{Name: name, Payload: payload})
}

func (m *Manager) Later(d time.Duration, name string, payload []byte) error {
	return m.q.Push(Job{Name: name, Payload: payload, Delay: d})
}

// Work pops jobs until ctx is cancelled.
func (m *Manager) Work(ctx context.Context) error {
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
		m.mu.RLock()
		h := m.handlers[job.Name]
		m.mu.RUnlock()
		if h == nil {
			continue
		}
		_ = h(ctx, job.Payload)
	}
}

// Memory is an in-process queue (useful in tests and single-node apps).
type Memory struct {
	ch chan delayed
}

type delayed struct {
	job Job
	at  time.Time
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

var errQueueFull = errString("gai/queue: full")

type errString string

func (e errString) Error() string { return string(e) }
