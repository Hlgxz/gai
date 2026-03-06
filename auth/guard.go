package auth

import (
	ghttp "github.com/Hlgxz/gai/http"
)

// Guard defines the contract for authentication drivers.
// Each guard (JWT, WeChat, etc.) implements this interface.
type Guard interface {
	// Name returns the guard identifier.
	Name() string

	// User extracts the authenticated user from the request context.
	// Returns nil if unauthenticated.
	User(c *ghttp.Context) any

	// Check returns true if the request is authenticated.
	Check(c *ghttp.Context) bool

	// Attempt tries to authenticate with the given credentials.
	Attempt(credentials map[string]any) (token string, err error)

	// Logout invalidates the current session/token if applicable.
	Logout(c *ghttp.Context) error
}
