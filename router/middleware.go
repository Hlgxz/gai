package router

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// Middleware is an alias for the handler function type used in the chain.
type Middleware = ghttp.HandlerFunc

// chain merges multiple middleware slices and a final handler into one slice,
// ready to be executed via Context.Next().
func chain(middlewares []ghttp.HandlerFunc, handler ghttp.HandlerFunc) []ghttp.HandlerFunc {
	final := make([]ghttp.HandlerFunc, 0, len(middlewares)+1)
	final = append(final, middlewares...)
	final = append(final, handler)
	return final
}
