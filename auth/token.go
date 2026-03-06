package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends jwt.RegisteredClaims with a user ID.
type Claims struct {
	jwt.RegisteredClaims
	UserID uint64         `json:"uid"`
	Extra  map[string]any `json:"ext,omitempty"`
}

// TokenConfig holds JWT signing parameters.
type TokenConfig struct {
	Secret string
	TTL    time.Duration // token lifetime
	Issuer string
}

// GenerateToken creates a signed JWT string.
func GenerateToken(cfg TokenConfig, userID uint64, extra map[string]any) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.TTL)),
		},
		UserID: userID,
		Extra:  extra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken validates and parses a JWT string.
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
	return claims, nil
}
