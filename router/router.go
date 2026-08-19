package router

import (
	"net"
	"net/http"
	"path/filepath"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
)

// Router is the core HTTP router with a segment trie, path parameters (:param),
// wildcards (*), named routes, and middleware chains.
type Router struct {
	routes           []*Route
	tree             *node
	named            map[string]*Route
	global           []ghttp.HandlerFunc
	notFound         ghttp.HandlerFunc
	methodNotAllowed ghttp.HandlerFunc
	trustAllProxies  bool
	trustedProxies   []*net.IPNet
}

type node struct {
	static   map[string]*node
	param    *node
	paramKey string
	wild     *node
	routes   map[string]*Route
}

func newNode() *node {
	return &node{static: make(map[string]*node), routes: make(map[string]*Route)}
}

// New creates a new Router.
func New() *Router {
	return &Router{
		tree:  newNode(),
		named: make(map[string]*Route),
	}
}

// Use adds global middleware applied to every route.
func (r *Router) Use(middlewares ...ghttp.HandlerFunc) *Router {
	r.global = append(r.global, middlewares...)
	return r
}

// SetTrustedProxies controls which remote IPs may supply X-Forwarded-For.
// Pass "*" to trust all (only behind a known reverse proxy).
func (r *Router) SetTrustedProxies(proxies []string) error {
	r.trustAllProxies = false
	r.trustedProxies = nil
	if len(proxies) == 1 && proxies[0] == "*" {
		r.trustAllProxies = true
		return nil
	}
	for _, p := range proxies {
		if p == "" {
			continue
		}
		if !strings.Contains(p, "/") {
			if strings.Contains(p, ":") {
				p += "/128"
			} else {
				p += "/32"
			}
		}
		_, network, err := net.ParseCIDR(p)
		if err != nil {
			return err
		}
		r.trustedProxies = append(r.trustedProxies, network)
	}
	return nil
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

// Head registers a HEAD route.
func (r *Router) Head(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("HEAD", pattern, handler, nil)
}

// Options registers an OPTIONS route.
func (r *Router) Options(pattern string, handler ghttp.HandlerFunc) *Route {
	return r.addRoute("OPTIONS", pattern, handler, nil)
}

// Any registers a route for all common HTTP methods.
func (r *Router) Any(pattern string, handler ghttp.HandlerFunc) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"} {
		r.addRoute(m, pattern, handler, nil)
	}
}

// Static serves files from dir under the given URL prefix.
func (r *Router) Static(prefix, dir string) {
	prefix = strings.TrimSuffix(prefix, "/")
	r.Get(prefix+"/*", func(c *ghttp.Context) {
		rest := c.Param("*")
		clean := filepath.Clean("/" + rest)
		c.File(filepath.Join(dir, strings.TrimPrefix(clean, "/")))
	})
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

// MethodNotAllowed sets a custom 405 handler.
func (r *Router) MethodNotAllowed(handler ghttp.HandlerFunc) {
	r.methodNotAllowed = handler
}

// Routes returns all registered routes (useful for debugging / listing).
func (r *Router) Routes() []*Route {
	return r.routes
}

// URL builds a path from a named route and parameter map.
func (r *Router) URL(name string, params map[string]string) string {
	rt, ok := r.named[name]
	if !ok {
		return ""
	}
	parts := segments(rt.Pattern)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			out = append(out, params[p[1:]])
			continue
		}
		if p == "*" {
			out = append(out, params["*"])
			continue
		}
		out = append(out, p)
	}
	return "/" + strings.Join(out, "/")
}

// ServeHTTP implements http.Handler, making the Router usable with net/http.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := ghttp.NewContext(w, req)
	c.SetProxyTrust(r.trustAllProxies, r.trustedProxies)

	route, params, allowed := r.lookup(req.Method, req.URL.Path)
	if route == nil {
		if len(allowed) > 0 {
			c.SetHeader("Allow", strings.Join(allowed, ", "))
			if r.methodNotAllowed != nil {
				c.SetHandlers(chain(r.global, r.methodNotAllowed))
				c.Next()
				return
			}
			c.JSON(http.StatusMethodNotAllowed, map[string]any{
				"code":    405,
				"message": "Method Not Allowed",
			})
			return
		}
		if r.notFound != nil {
			c.SetHandlers(chain(r.global, r.notFound))
			c.Next()
		} else {
			http.NotFound(w, req)
		}
		return
	}

	c.Params = params
	if req.Method == http.MethodHead && route.Method == "GET" {
		c.Writer = headWriter{ResponseWriter: w}
	}

	allMw := make([]ghttp.HandlerFunc, 0, len(r.global)+len(route.Middlewares)+1)
	allMw = append(allMw, r.global...)
	allMw = append(allMw, route.Middlewares...)
	allMw = append(allMw, route.Handler)

	c.SetHandlers(allMw)
	c.Next()
}

type headWriter struct {
	http.ResponseWriter
}

func (w headWriter) Write(b []byte) (int, error) { return len(b), nil }

func (r *Router) addRoute(method, pattern string, handler ghttp.HandlerFunc, mw []ghttp.HandlerFunc) *Route {
	route := &Route{
		Method:      method,
		Pattern:     pattern,
		Handler:     handler,
		Middlewares: mw,
		segs:        segments(pattern),
		router:      r,
	}
	r.routes = append(r.routes, route)
	r.tree.insert(route.segs, method, route)
	return route
}

func (r *Router) lookup(method, path string) (*Route, map[string]string, []string) {
	return r.tree.lookup(segments(path), method)
}

func (n *node) insert(segs []string, method string, r *Route) {
	cur := n
	for i, seg := range segs {
		if seg == "*" {
			if cur.wild == nil {
				cur.wild = newNode()
			}
			cur = cur.wild
			_ = i
			break
		}
		if strings.HasPrefix(seg, ":") {
			if cur.param == nil {
				cur.param = newNode()
				cur.paramKey = seg[1:]
			}
			cur = cur.param
			continue
		}
		next, ok := cur.static[seg]
		if !ok {
			next = newNode()
			cur.static[seg] = next
		}
		cur = next
	}
	if cur.routes == nil {
		cur.routes = make(map[string]*Route)
	}
	cur.routes[method] = r
}

func (n *node) lookup(segs []string, method string) (*Route, map[string]string, []string) {
	params := make(map[string]string)
	cur := n
	for i, seg := range segs {
		if next, ok := cur.static[seg]; ok {
			cur = next
			continue
		}
		if cur.param != nil {
			params[cur.paramKey] = seg
			cur = cur.param
			continue
		}
		if cur.wild != nil {
			params["*"] = strings.Join(segs[i:], "/")
			cur = cur.wild
			break
		}
		return nil, nil, nil
	}

	if cur.routes == nil || len(cur.routes) == 0 {
		return nil, nil, nil
	}
	if rt, ok := cur.routes[method]; ok {
		return rt, params, nil
	}
	if method == http.MethodHead {
		if rt, ok := cur.routes["GET"]; ok {
			return rt, params, nil
		}
	}
	allowed := make([]string, 0, len(cur.routes))
	for m := range cur.routes {
		allowed = append(allowed, m)
	}
	return nil, params, allowed
}
