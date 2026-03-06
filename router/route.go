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
	segs        []string // cached segments from Pattern
}

// segments splits a URL pattern into its path parts.
func segments(pattern string) []string {
	return splitPath(pattern)
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
