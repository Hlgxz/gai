package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

const csrfCookie = "gai_csrf"
const csrfHeader = "X-CSRF-Token"

// CSRF implements the double-submit cookie pattern. Safe methods set a token;
// mutating methods must present the same value in the header or `_token` form field.
func CSRF(secret string) ghttp.HandlerFunc {
	return func(c *ghttp.Context) {
		token, _ := c.Cookie(csrfCookie)
		if token == "" {
			token = newCSRFToken(secret)
			c.SetCookie(&http.Cookie{
				Name:     csrfCookie,
				Value:    token,
				Path:     "/",
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int((12 * time.Hour).Seconds()),
			})
		}

		c.Set("csrf_token", token)

		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}

		got := c.Header(csrfHeader)
		if got == "" {
			got = c.PostForm("_token")
		}
		if got == "" || !hmac.Equal([]byte(got), []byte(token)) {
			c.AbortWithJSON(http.StatusForbidden, map[string]any{
				"code":    403,
				"message": "CSRF token mismatch",
			})
			return
		}
		c.Next()
	}
}

func newCSRFToken(secret string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(b)
	raw := append(b, mac.Sum(nil)...)
	return hex.EncodeToString(raw)
}

// CSRFToken returns the token stored by CSRF middleware.
func CSRFToken(c *ghttp.Context) string {
	if v, ok := c.Get("csrf_token"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
