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

// NewJWTGuard creates a JWT guard with the given secret and TTL.
func NewJWTGuard(secret string, ttlSeconds int) *JWTGuard {
	return &JWTGuard{
		config: TokenConfig{
			Secret: secret,
			TTL:    time.Duration(ttlSeconds) * time.Second,
			Issuer: "gai",
		},
	}
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
	return g.extractClaims(c) != nil
}

// Attempt is not directly applicable for JWT (no session). Use GenerateToken.
func (g *JWTGuard) Attempt(credentials map[string]any) (string, error) {
	uid, _ := credentials["user_id"].(uint64)
	extra, _ := credentials["extra"].(map[string]any)
	return GenerateToken(g.config, uid, extra)
}

func (g *JWTGuard) Logout(_ *ghttp.Context) error {
	return nil
}

// IssueToken generates a new JWT for the given user ID.
func (g *JWTGuard) IssueToken(userID uint64, extra map[string]any) (string, error) {
	return GenerateToken(g.config, userID, extra)
}

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
