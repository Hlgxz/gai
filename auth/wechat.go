package auth

import (
	"fmt"
	"hash/fnv"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

// MiniProgramSession is the result of exchanging a WeChat login code.
type MiniProgramSession struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// Code2Session exchanges a WeChat js_code for openid.
type Code2Session func(code string) (*MiniProgramSession, error)

// WeChatGuard authenticates mini-program clients: Attempt exchanges a code
// and issues a JWT carrying openid.
type WeChatGuard struct {
	jwt *JWTGuard
	c2s Code2Session
}

func NewWeChatGuard(secret string, ttlSeconds int, c2s Code2Session) *WeChatGuard {
	g := NewJWTGuard(secret, ttlSeconds)
	return &WeChatGuard{jwt: g, c2s: c2s}
}

func (g *WeChatGuard) SetRevoker(r Revoker) { g.jwt.SetRevoker(r) }

func (g *WeChatGuard) SetRefreshTTL(d time.Duration) { g.jwt.SetRefreshTTL(d) }

func (g *WeChatGuard) Name() string { return "wechat" }

func (g *WeChatGuard) User(c *ghttp.Context) any { return g.jwt.User(c) }

func (g *WeChatGuard) Check(c *ghttp.Context) bool { return g.jwt.Check(c) }

func (g *WeChatGuard) Attempt(credentials map[string]any) (string, error) {
	if g.c2s == nil {
		return "", fmt.Errorf("gai/auth: wechat guard has no Code2Session")
	}
	code, _ := credentials["code"].(string)
	if code == "" {
		return "", fmt.Errorf("gai/auth: missing wechat code")
	}
	sess, err := g.c2s(code)
	if err != nil {
		return "", err
	}
	extra := map[string]any{
		"openid":  sess.OpenID,
		"unionid": sess.UnionID,
	}
	return g.jwt.IssueToken(openidToID(sess.OpenID), extra)
}

func (g *WeChatGuard) Logout(c *ghttp.Context) error { return g.jwt.Logout(c) }

func (g *WeChatGuard) IssueToken(userID uint64, extra map[string]any) (string, error) {
	return g.jwt.IssueToken(userID, extra)
}

func openidToID(openid string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(openid))
	return h.Sum64()
}
