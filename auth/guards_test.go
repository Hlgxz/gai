package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hlgxz/gai/auth"
	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/session"
)

func TestSessionGuard(t *testing.T) {
	g := auth.NewSessionGuard("session", func(credentials map[string]any) (any, map[string]any, error) {
		if credentials["password"] != "secret" {
			return nil, nil, errAuth
		}
		return uint64(9), nil, nil
	})
	if g.Name() != "session" {
		t.Fatal(g.Name())
	}

	store := session.NewMemoryStore()
	mgr := session.New(store)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := ghttp.NewContext(rec, req)
	mgr.Middleware()(c) // this calls Next which is empty

	// Manually load session like middleware does
	c2rec := httptest.NewRecorder()
	c2req := httptest.NewRequest(http.MethodPost, "/login", nil)
	c2 := ghttp.NewContext(c2rec, c2req)
	id := "sess1"
	c2.SetCookie(&http.Cookie{Name: mgr.CookieName, Value: id})
	c2.Set(session.ContextKey, &session.Session{ID: id, Values: map[string]any{}})

	if g.Check(c2) {
		t.Fatal("should be anonymous")
	}
	if err := g.Login(c2, map[string]any{"password": "secret"}); err != nil {
		t.Fatal(err)
	}
	if !g.Check(c2) {
		t.Fatal("should be logged in")
	}
	if g.User(c2) == nil {
		t.Fatal("user")
	}
	_ = g.Logout(c2)
}

func TestWeChatGuard(t *testing.T) {
	g := auth.NewWeChatGuard("secret-key-for-wechat-tests", 60, func(code string) (*auth.MiniProgramSession, error) {
		if code != "ok" {
			return nil, errAuth
		}
		return &auth.MiniProgramSession{OpenID: "oid", UnionID: "uid"}, nil
	})
	tok, err := g.Attempt(map[string]any{"code": "ok"})
	if err != nil || tok == "" {
		t.Fatalf("%v %s", err, tok)
	}
}

type errAuthT string

func (e errAuthT) Error() string { return string(e) }

var errAuth error = errAuthT("bad credentials")
