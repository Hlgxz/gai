package auth

import (
	"strings"
	"time"

	ghttp "github.com/Hlgxz/gai/http"
)

// JWTGuard implements the Guard interface using JSON Web Tokens.
type JWTGuard struct {
	config TokenConfig
}

// NewJWTGuard creates a JWT guard with the given secret and access-token TTL (seconds).
func NewJWTGuard(secret string, ttlSeconds int) *JWTGuard {
	return &JWTGuard{
		config: TokenConfig{
			Secret:     secret,
			TTL:        time.Duration(ttlSeconds) * time.Second,
			RefreshTTL: time.Duration(ttlSeconds) * time.Second * 24,
			Issuer:     "gai",
			Revoker:    NewMemoryRevoker(),
		},
	}
}

// SetRevoker replaces the token blacklist implementation (e.g. Redis).
func (g *JWTGuard) SetRevoker(r Revoker) {
	g.config.Revoker = r
}

// SetRefreshTTL overrides the refresh token lifetime.
func (g *JWTGuard) SetRefreshTTL(d time.Duration) {
	g.config.RefreshTTL = d
}

func (g *JWTGuard) Name() string { return "jwt" }

func (g *JWTGuard) User(c *ghttp.Context) any {
	claims := g.extractClaims(c)
	if claims == nil {
		return nil
	}
	return claims
}

func (g *JWTGuard) Check(c *ghttp.Context) bool {
	claims := g.extractClaims(c)
	if claims == nil {
		return false
	}
	return claims.TokenType == "" || claims.TokenType == TokenTypeAccess
}

// Attempt is not directly applicable for JWT (no session). Use IssueTokenPair.
func (g *JWTGuard) Attempt(credentials map[string]any) (string, error) {
	uid, _ := credentials["user_id"].(uint64)
	extra, _ := credentials["extra"].(map[string]any)
	return GenerateToken(g.config, uid, extra)
}

func (g *JWTGuard) Logout(c *ghttp.Context) error {
	claims := g.extractClaims(c)
	if claims == nil || g.config.Revoker == nil {
		return nil
	}
	until := time.Now().Add(g.config.TTL)
	if claims.ExpiresAt != nil {
		until = claims.ExpiresAt.Time
	}
	g.config.Revoker.Revoke(claims.ID, until)
	return nil
}

// IssueToken generates a new access JWT for the given user ID.
func (g *JWTGuard) IssueToken(userID uint64, extra map[string]any) (string, error) {
	return GenerateToken(g.config, userID, extra)
}

// IssueTokenPair generates access + refresh tokens.
func (g *JWTGuard) IssueTokenPair(userID uint64, extra map[string]any) (*TokenPair, error) {
	return GenerateTokenPair(g.config, userID, extra)
}

// Refresh exchanges a valid refresh token for a new token pair.
// The old refresh token is revoked.
func (g *JWTGuard) Refresh(refreshToken string) (*TokenPair, error) {
	claims, err := ParseToken(g.config, refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, errRefreshType
	}
	if g.config.Revoker != nil && claims.ID != "" {
		until := time.Now().Add(g.config.refreshTTL())
		if claims.ExpiresAt != nil {
			until = claims.ExpiresAt.Time
		}
		g.config.Revoker.Revoke(claims.ID, until)
	}
	return GenerateTokenPair(g.config, claims.UserID, claims.Extra)
}

var errRefreshType = errString("gai/auth: not a refresh token")

type errString string

func (e errString) Error() string { return string(e) }

// ParseFromRequest extracts and validates the JWT from the Authorization header.
func (g *JWTGuard) ParseFromRequest(c *ghttp.Context) (*Claims, error) {
	tokenStr := extractBearerToken(c)
	if tokenStr == "" {
		return nil, nil
	}
	return ParseToken(g.config, tokenStr)
}

func (g *JWTGuard) extractClaims(c *ghttp.Context) *Claims {
	if cached, ok := c.Get("auth_claims"); ok {
		return cached.(*Claims)
	}
	claims, err := g.ParseFromRequest(c)
	if err != nil || claims == nil {
		return nil
	}
	c.Set("auth_claims", claims)
	c.Set("auth_user_id", claims.UserID)
	return claims
}

func extractBearerToken(c *ghttp.Context) string {
	auth := c.Header("Authorization")
	if auth == "" {
		return c.Query("token")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
