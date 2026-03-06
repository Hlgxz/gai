package router

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// chain merges multiple middleware slices and a final handler into one slice,
// ready to be executed via Context.Next().
func chain(middlewares []ghttp.HandlerFunc, handler ghttp.HandlerFunc) []ghttp.HandlerFunc {
	final := make([]ghttp.HandlerFunc, 0, len(middlewares)+1)
	final = append(final, middlewares...)
	final = append(final, handler)
	return final
}
