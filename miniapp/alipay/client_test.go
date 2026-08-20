package alipay_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Hlgxz/gai/miniapp/alipay"
)

func TestSystemOauthTokenSigned(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	var sawSign bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "sign=") || !strings.Contains(string(body), "sign_type=RSA2") {
			t.Errorf("unsigned request: %s", body)
		}
		sawSign = strings.Contains(string(body), "sign=")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alipay_system_oauth_token_response":{"access_token":"tok","user_id":"u1","expires_in":3600}}`))
	}))
	defer srv.Close()

	c := alipay.NewClient(alipay.Config{
		AppID:      "app",
		PrivateKey: string(pemKey),
		Gateway:    srv.URL,
	})
	tok, err := c.SystemOauthToken("authcode")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "tok" || tok.UserID != "u1" {
		t.Fatalf("%+v", tok)
	}
	if !sawSign {
		t.Fatal("expected RSA2 sign in request")
	}
}
