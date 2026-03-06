package router

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// Route represents a single registered route with its method, pattern,
// handler, and middleware stack.
type Route struct {
	Method      string
	Pattern     string
	Handler     ghttp.HandlerFunc
	Middlewares []ghttp.HandlerFunc
	Name        string
}

// segments splits a URL pattern into its path parts, filtering out empties.
func segments(pattern string) []string {
	var parts []string
	for _, s := range splitPath(pattern) {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

func splitPath(path string) []string {
	result := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		result = append(result, path[start:])
	}
	return result
}
