package event

import "sync"

// Listener handles a dispatched event.
type Listener func(payload any)

// Dispatcher is a simple in-process event bus.
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
}

// New creates an event dispatcher.
func New() *Dispatcher {
	return &Dispatcher{listeners: make(map[string][]Listener)}
}

// Listen registers a listener for an event name.
func (d *Dispatcher) Listen(name string, l Listener) {
	d.mu.Lock()
	d.listeners[name] = append(d.listeners[name], l)
	d.mu.Unlock()
}

// Dispatch notifies all listeners. Listeners run synchronously in registration order.
func (d *Dispatcher) Dispatch(name string, payload any) {
	d.mu.RLock()
	ls := append([]Listener(nil), d.listeners[name]...)
	d.mu.RUnlock()
	for _, l := range ls {
		l(payload)
	}
}

// Forget removes all listeners for an event.
func (d *Dispatcher) Forget(name string) {
	d.mu.Lock()
	delete(d.listeners, name)
	d.mu.Unlock()
}
