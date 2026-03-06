package router

import (
	"net/http"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
)

// Router is the core HTTP router with method-bucketed matching, supporting
// path parameters (:param) and middleware chains.
type Router struct {
	routes     []*Route
	byMethod   map[string][]*Route
	global     []ghttp.HandlerFunc
	notFound   ghttp.HandlerFunc
}

// New creates a new Router.
func New() *Router {
	return &Router{
		byMethod: make(map[string][]*Route),
	}
}

// Use adds global middleware applied to every route.
func (r *Router) Use(middlewares ...ghttp.HandlerFunc) *Router {
	r.global = append(r.global, middlewares...)
	return r
}

// Get registers a GET route at the top level.
func (r *Router) Get(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("GET", pattern, handler, nil)
}

// Post registers a POST route.
func (r *Router) Post(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("POST", pattern, handler, nil)
}

// Put registers a PUT route.
func (r *Router) Put(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("PUT", pattern, handler, nil)
}

// Patch registers a PATCH route.
func (r *Router) Patch(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("PATCH", pattern, handler, nil)
}

// Delete registers a DELETE route.
func (r *Router) Delete(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("DELETE", pattern, handler, nil)
}

// Any registers a route for all common HTTP methods.
func (r *Router) Any(pattern string, handler ghttp.HandlerFunc) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"} {
		r.addRoute(m, pattern, handler, nil)
	}
}

// Group creates a route group with a shared prefix and optional middleware.
func (r *Router) Group(prefix string, fn func(g *Group)) {
	g := &Group{
		prefix: prefix,
		router: r,
	}
	fn(g)
}

// Resource registers a RESTful resource at the top level.
func (r *Router) Resource(prefix string, ctrl ResourceController) {
	r.Get(prefix, ctrl.Index)
	r.Post(prefix, ctrl.Store)
	r.Get(prefix+"/:id", ctrl.Show)
	r.Put(prefix+"/:id", ctrl.Update)
	r.Delete(prefix+"/:id", ctrl.Destroy)
}

// NotFound sets a custom 404 handler.
func (r *Router) NotFound(handler ghttp.HandlerFunc) {
	r.notFound = handler
}

// Routes returns all registered routes (useful for debugging / listing).
func (r *Router) Routes() []*Route {
	return r.routes
}

// ServeHTTP implements http.Handler, making the Router usable with net/http.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := ghttp.NewContext(w, req)

	route, params := r.match(req.Method, req.URL.Path)
	if route == nil {
		if r.notFound != nil {
			allMw := chain(r.global, r.notFound)
			c.SetHandlers(allMw)
			c.Next()
		} else {
			http.NotFound(w, req)
		}
		return
	}

	c.Params = params

	// Build the full handler chain: global middleware -> route middleware -> handler.
	allMw := make([]ghttp.HandlerFunc, 0, len(r.global)+len(route.Middlewares)+1)
	allMw = append(allMw, r.global...)
	allMw = append(allMw, route.Middlewares...)
	allMw = append(allMw, route.Handler)

	c.SetHandlers(allMw)
	c.Next()
}

func (r *Router) addRoute(method, pattern string, handler ghttp.HandlerFunc, mw []ghttp.HandlerFunc) *Route {
	route := &Route{
		Method:      method,
		Pattern:     pattern,
		Handler:     handler,
		Middlewares: mw,
		segs:        segments(pattern),
	}
	r.routes = append(r.routes, route)
	r.byMethod[method] = append(r.byMethod[method], route)
	return route
}

// match finds the first route that matches the given method and path,
// extracting path parameters. Routes are bucketed by method to avoid
// scanning the entire list.
func (r *Router) match(method, path string) (*Route, map[string]string) {
	bucket := r.byMethod[method]
	if len(bucket) == 0 {
		return nil, nil
	}

	reqSegs := segments(path)

	for _, route := range bucket {
		if params, ok := matchSegments(route.segs, reqSegs); ok {
			return route, params
		}
	}
	return nil, nil
}

// matchSegments compares route pattern segments against request segments,
// extracting :param values.
func matchSegments(routeSegs, reqSegs []string) (map[string]string, bool) {
	if len(routeSegs) != len(reqSegs) {
		// Support wildcard * as the last segment to match remaining path.
		if len(routeSegs) > 0 && routeSegs[len(routeSegs)-1] == "*" {
			if len(reqSegs) < len(routeSegs)-1 {
				return nil, false
			}
			params := make(map[string]string)
			for i := 0; i < len(routeSegs)-1; i++ {
				if strings.HasPrefix(routeSegs[i], ":") {
					params[routeSegs[i][1:]] = reqSegs[i]
				} else if routeSegs[i] != reqSegs[i] {
					return nil, false
				}
			}
			params["*"] = strings.Join(reqSegs[len(routeSegs)-1:], "/")
			return params, true
		}
		return nil, false
	}

	params := make(map[string]string)
	for i, seg := range routeSegs {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = reqSegs[i]
		} else if seg != reqSegs[i] {
			return nil, false
		}
	}
	return params, true
}
