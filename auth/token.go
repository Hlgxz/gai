package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims extends jwt.RegisteredClaims with a user ID and token type.
type Claims struct {
	jwt.RegisteredClaims
	UserID    uint64         `json:"uid"`
	TokenType string         `json:"typ,omitempty"`
	Extra     map[string]any `json:"ext,omitempty"`
}

// TokenConfig holds JWT signing parameters.
type TokenConfig struct {
	Secret     string
	TTL        time.Duration // access token lifetime
	RefreshTTL time.Duration // refresh token lifetime
	Issuer     string
	Revoker    Revoker
}

func (cfg TokenConfig) refreshTTL() time.Duration {
	if cfg.RefreshTTL > 0 {
		return cfg.RefreshTTL
	}
	if cfg.TTL > 0 {
		return cfg.TTL * 24
	}
	return 7 * 24 * time.Hour
}

func newJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateToken creates a signed access JWT.
func GenerateToken(cfg TokenConfig, userID uint64, extra map[string]any) (string, error) {
	return generateToken(cfg, userID, TokenTypeAccess, cfg.TTL, extra)
}

// GenerateRefreshToken creates a signed refresh JWT.
func GenerateRefreshToken(cfg TokenConfig, userID uint64, extra map[string]any) (string, error) {
	return generateToken(cfg, userID, TokenTypeRefresh, cfg.refreshTTL(), extra)
}

func generateToken(cfg TokenConfig, userID uint64, typ string, ttl time.Duration, extra map[string]any) (string, error) {
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        newJTI(),
		},
		UserID:    userID,
		TokenType: typ,
		Extra:     extra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken validates and parses a JWT string. Revoked tokens are rejected.
func ParseToken(cfg TokenConfig, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("gai/auth: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("gai/auth: invalid token claims")
	}
	if cfg.Revoker != nil && claims.ID != "" && cfg.Revoker.Revoked(claims.ID) {
		return nil, fmt.Errorf("gai/auth: token revoked")
	}
	return claims, nil
}

// TokenPair is an access + refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GenerateTokenPair issues both access and refresh tokens.
func GenerateTokenPair(cfg TokenConfig, userID uint64, extra map[string]any) (*TokenPair, error) {
	access, err := GenerateToken(cfg, userID, extra)
	if err != nil {
		return nil, err
	}
	refresh, err := GenerateRefreshToken(cfg, userID, extra)
	if err != nil {
		return nil, err
	}
	exp := cfg.TTL
	if exp <= 0 {
		exp = 2 * time.Hour
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(exp.Seconds()),
		TokenType:    "Bearer",
	}, nil
}
