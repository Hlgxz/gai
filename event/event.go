package event

import "sync"

// Listener handles a dispatched event.
type Listener func(payload any)

type listener struct {
	fn    Listener
	async bool
}

// Dispatcher is an in-process event bus.
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]listener
}

// New creates an event dispatcher.
func New() *Dispatcher {
	return &Dispatcher{listeners: make(map[string][]listener)}
}

// Listen registers a synchronous listener for an event name.
func (d *Dispatcher) Listen(name string, l Listener) {
	d.mu.Lock()
	d.listeners[name] = append(d.listeners[name], listener{fn: l})
	d.mu.Unlock()
}

// ListenAsync registers a listener that runs in its own goroutine.
func (d *Dispatcher) ListenAsync(name string, l Listener) {
	d.mu.Lock()
	d.listeners[name] = append(d.listeners[name], listener{fn: l, async: true})
	d.mu.Unlock()
}

// Dispatch notifies all listeners. Synchronous listeners run in registration
// order; panics in a listener are recovered so others still run.
func (d *Dispatcher) Dispatch(name string, payload any) {
	d.mu.RLock()
	ls := append([]listener(nil), d.listeners[name]...)
	d.mu.RUnlock()
	for _, l := range ls {
		if l.async {
			go safeCall(l.fn, payload)
			continue
		}
		safeCall(l.fn, payload)
	}
}

func safeCall(fn Listener, payload any) {
	defer func() { _ = recover() }()
	fn(payload)
}

// Forget removes all listeners for an event.
func (d *Dispatcher) Forget(name string) {
	d.mu.Lock()
	delete(d.listeners, name)
	d.mu.Unlock()
}
