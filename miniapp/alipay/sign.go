package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func (c *Client) signParams(params map[string]string) (string, error) {
	key, err := parsePrivateKey(c.config.PrivateKey)
	if err != nil {
		return "", err
	}
	src := encodeAlipayParams(params)
	sum := sha256.Sum256([]byte(src))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("alipay: sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func encodeAlipayParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

func verifyAlipaySign(publicKey, content, sign string) error {
	pub, err := parsePublicKey(publicKey)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], raw)
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode([]byte(ensurePEM(raw, "PRIVATE KEY")))
	if block == nil {
		block, _ = pem.Decode([]byte(ensurePEM(string(rest), "RSA PRIVATE KEY")))
	}
	if block == nil {
		der, err := base64.StdEncoding.DecodeString(compactKey(raw))
		if err != nil {
			return nil, fmt.Errorf("alipay: invalid private key")
		}
		return parsePrivateDER(der)
	}
	return parsePrivateDER(block.Bytes)
}

func parsePrivateDER(der []byte) (*rsa.PrivateKey, error) {
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("alipay: not an RSA private key")
		}
		return rk, nil
	}
	k, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("alipay: parse private key: %w", err)
	}
	return k, nil
}

func parsePublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(ensurePEM(raw, "PUBLIC KEY")))
	var der []byte
	if block != nil {
		der = block.Bytes
	} else {
		var err error
		der, err = base64.StdEncoding.DecodeString(compactKey(raw))
		if err != nil {
			return nil, fmt.Errorf("alipay: invalid public key")
		}
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("alipay: parse public key: %w", err)
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("alipay: not an RSA public key")
	}
	return rk, nil
}

func ensurePEM(raw, kind string) string {
	if strings.Contains(raw, "BEGIN") {
		return raw
	}
	body := chunk64(compactKey(raw))
	return "-----BEGIN " + kind + "-----\n" + body + "\n-----END " + kind + "-----"
}

func compactKey(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func chunk64(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 64 {
		end := i + 64
		if end > len(s) {
			end = len(s)
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s[i:end])
	}
	return b.String()
}

func formValues(params map[string]string) url.Values {
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	return v
}
