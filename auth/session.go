package auth

import (
	"fmt"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/session"
)

const sessionUserKey = "user_id"

// Authenticator verifies credentials and returns a user id (and optional extra claims).
type Authenticator func(credentials map[string]any) (userID any, extra map[string]any, err error)

// SessionGuard stores the authenticated user id in a cookie session.
type SessionGuard struct {
	name  string
	login Authenticator
}

func NewSessionGuard(name string, login Authenticator) *SessionGuard {
	if name == "" {
		name = "session"
	}
	return &SessionGuard{name: name, login: login}
}

func (g *SessionGuard) Name() string { return g.name }

func (g *SessionGuard) User(c *ghttp.Context) any {
	if v, ok := c.Get("auth_user"); ok {
		return v
	}
	sess := session.FromContext(c)
	if sess == nil {
		return nil
	}
	return sess.Get(sessionUserKey)
}

func (g *SessionGuard) Check(c *ghttp.Context) bool {
	sess := session.FromContext(c)
	if sess == nil {
		return false
	}
	id := sess.Get(sessionUserKey)
	if id == nil || fmt.Sprint(id) == "" {
		return false
	}
	c.Set("auth_user_id", id)
	c.Set("auth_user", id)
	return true
}

func (g *SessionGuard) Attempt(credentials map[string]any) (string, error) {
	if g.login == nil {
		return "", fmt.Errorf("gai/auth: session guard has no authenticator")
	}
	id, _, err := g.login(credentials)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(id), nil
}

func (g *SessionGuard) Logout(c *ghttp.Context) error {
	sess := session.FromContext(c)
	if sess == nil {
		return nil
	}
	sess.Forget(sessionUserKey)
	return nil
}

// Login authenticates and writes user_id into the request session.
func (g *SessionGuard) Login(c *ghttp.Context, credentials map[string]any) error {
	if g.login == nil {
		return fmt.Errorf("gai/auth: session guard has no authenticator")
	}
	sess := session.FromContext(c)
	if sess == nil {
		return fmt.Errorf("gai/auth: session middleware not installed")
	}
	id, extra, err := g.login(credentials)
	if err != nil {
		return err
	}
	sess.Set(sessionUserKey, id)
	c.Set("auth_user_id", id)
	c.Set("auth_user", id)
	if extra != nil {
		c.Set("auth_extra", extra)
	}
	return nil
}
