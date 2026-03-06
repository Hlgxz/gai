package gai

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is a lightweight dependency injection container inspired by
// Laravel's Service Container. It supports both transient (Bind) and
// singleton (Singleton) bindings, resolved lazily on first use.
type Container struct {
	mu         sync.RWMutex
	bindings   map[string]binding
	instances  map[string]any
	aliases    map[reflect.Type]string
}

type binding struct {
	resolver  func(c *Container) any
	singleton bool
}

func newContainer() *Container {
	return &Container{
		bindings:  make(map[string]binding),
		instances: make(map[string]any),
		aliases:   make(map[reflect.Type]string),
	}
}

// Bind registers a transient factory. Each call to Make produces a new instance.
func (c *Container) Bind(name string, resolver func(c *Container) any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings[name] = binding{resolver: resolver, singleton: false}
}

// Singleton registers a factory that is resolved once and cached forever.
func (c *Container) Singleton(name string, resolver func(c *Container) any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings[name] = binding{resolver: resolver, singleton: true}
}

// Instance binds an already-constructed value as a singleton.
func (c *Container) Instance(name string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[name] = value
}

// Alias associates a Go type with a binding name so Make can resolve by type.
func (c *Container) Alias(typ reflect.Type, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[typ] = name
}

// Resolve retrieves or creates the value bound to name.
func (c *Container) Resolve(name string) (any, error) {
	c.mu.RLock()
	if inst, ok := c.instances[name]; ok {
		c.mu.RUnlock()
		return inst, nil
	}
	b, ok := c.bindings[name]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("gai: no binding found for [%s]", name)
	}

	instance := b.resolver(c)

	if b.singleton {
		c.mu.Lock()
		c.instances[name] = instance
		c.mu.Unlock()
	}

	return instance, nil
}

// MustResolve is like Resolve but panics on failure.
func (c *Container) MustResolve(name string) any {
	v, err := c.Resolve(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Has returns true if a binding or instance exists for the given name.
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.instances[name]; ok {
		return true
	}
	_, ok := c.bindings[name]
	return ok
}

// Make is a generic helper to resolve and type-assert from the container.
func Make[T any](c *Container, name string) T {
	v := c.MustResolve(name)
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("gai: binding [%s] is %T, not %s", name, v, reflect.TypeFor[T]()))
	}
	return t
}
