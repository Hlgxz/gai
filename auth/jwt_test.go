package auth_test

import (
	"testing"
	"time"

	"github.com/Hlgxz/gai/auth"
)

func TestJWTPairRefreshAndRevoke(t *testing.T) {
	g := auth.NewJWTGuard("secret-key-for-test", 60)
	pair, err := g.IssueTokenPair(7, map[string]any{"role": "admin"})
	if err != nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("pair: %+v %v", pair, err)
	}

	claims, err := auth.ParseToken(auth.TokenConfig{Secret: "secret-key-for-test", TTL: time.Minute, Revoker: auth.NewMemoryRevoker()}, pair.AccessToken)
	if err != nil || claims.UserID != 7 {
		t.Fatalf("parse access: %+v %v", claims, err)
	}

	next, err := g.Refresh(pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Refresh(pair.RefreshToken); err == nil {
		t.Fatal("old refresh should be revoked")
	}
	if next.AccessToken == "" {
		t.Fatal("new pair empty")
	}
}
