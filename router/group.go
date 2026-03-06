package router

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// Group allows defining routes under a common prefix with shared middleware,
// similar to Laravel's Route::group().
type Group struct {
	prefix      string
	middlewares []ghttp.HandlerFunc
	router      *Router
}

// Use appends middleware to this group.
func (g *Group) Use(middlewares ...ghttp.HandlerFunc) *Group {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// Group creates a nested sub-group.
func (g *Group) Group(prefix string, fn func(sub *Group)) {
	sub := &Group{
		prefix:      g.prefix + prefix,
		middlewares: copyMiddlewares(g.middlewares),
		router:      g.router,
	}
	fn(sub)
}

// Get registers a GET route.
func (g *Group) Get(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("GET", pattern, handler)
}

// Post registers a POST route.
func (g *Group) Post(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("POST", pattern, handler)
}

// Put registers a PUT route.
func (g *Group) Put(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("PUT", pattern, handler)
}

// Patch registers a PATCH route.
func (g *Group) Patch(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("PATCH", pattern, handler)
}

// Delete registers a DELETE route.
func (g *Group) Delete(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("DELETE", pattern, handler)
}

// Options registers an OPTIONS route.
func (g *Group) Options(pattern string, handler ghttp.HandlerFunc) *Route {
	return g.addRoute("OPTIONS", pattern, handler)
}

// Any registers a route for all common HTTP methods.
func (g *Group) Any(pattern string, handler ghttp.HandlerFunc) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"} {
		g.addRoute(m, pattern, handler)
	}
}

// ResourceController defines the interface for RESTful resource controllers.
type ResourceController interface {
	Index(c *ghttp.Context)
	Show(c *ghttp.Context)
	Store(c *ghttp.Context)
	Update(c *ghttp.Context)
	Destroy(c *ghttp.Context)
}

// Resource registers a full RESTful resource (index, show, store, update, destroy).
func (g *Group) Resource(prefix string, ctrl ResourceController) {
	g.Get(prefix, ctrl.Index)
	g.Post(prefix, ctrl.Store)
	g.Get(prefix+"/:id", ctrl.Show)
	g.Put(prefix+"/:id", ctrl.Update)
	g.Delete(prefix+"/:id", ctrl.Destroy)
}

func (g *Group) addRoute(method, pattern string, handler ghttp.HandlerFunc) *Route {
	fullPattern := g.prefix + pattern
	route := &Route{
		Method:      method,
		Pattern:     fullPattern,
		Handler:     handler,
		Middlewares: copyMiddlewares(g.middlewares),
	}
	g.router.routes = append(g.router.routes, route)
	return route
}

func copyMiddlewares(src []ghttp.HandlerFunc) []ghttp.HandlerFunc {
	if len(src) == 0 {
		return nil
	}
	dst := make([]ghttp.HandlerFunc, len(src))
	copy(dst, src)
	return dst
}
