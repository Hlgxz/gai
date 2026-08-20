package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// S3 is an S3-compatible disk (AWS S3, MinIO, OSS with S3 API).
type S3 struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	BaseURL   string
	PathStyle bool
	Client    *http.Client
	Now       func() time.Time
}

func (s *S3) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}

func (s *S3) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *S3) region() string {
	if s.Region != "" {
		return s.Region
	}
	return "us-east-1"
}

func (s *S3) host() string {
	ep := strings.TrimPrefix(strings.TrimPrefix(s.Endpoint, "https://"), "http://")
	ep = strings.TrimSuffix(ep, "/")
	if ep == "" {
		if s.PathStyle {
			return "s3." + s.region() + ".amazonaws.com"
		}
		return s.Bucket + ".s3." + s.region() + ".amazonaws.com"
	}
	if s.PathStyle {
		return ep
	}
	return s.Bucket + "." + ep
}

func (s *S3) objectPath(path string) string {
	p := strings.TrimPrefix(path, "/")
	if s.PathStyle {
		return "/" + s.Bucket + "/" + p
	}
	return "/" + p
}

func (s *S3) scheme() string {
	if strings.HasPrefix(s.Endpoint, "http://") {
		return "http"
	}
	return "https"
}

func (s *S3) Put(path string, r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	req, err := s.newRequest(http.MethodPut, path, body)
	if err != nil {
		return err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gai/storage/s3: put %s: %s %s", path, resp.Status, b)
	}
	return nil
}

func (s *S3) Get(path string) (io.ReadCloser, error) {
	req, err := s.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("gai/storage/s3: get %s: %s %s", path, resp.Status, b)
	}
	return resp.Body, nil
}

func (s *S3) Delete(path string) error {
	req, err := s.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gai/storage/s3: delete %s: %s %s", path, resp.Status, b)
	}
	return nil
}

func (s *S3) Exists(path string) (bool, error) {
	req, err := s.newRequest(http.MethodHead, path, nil)
	if err != nil {
		return false, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("gai/storage/s3: head %s: %s", path, resp.Status)
	}
	return true, nil
}

func (s *S3) URL(path string) string {
	p := strings.TrimPrefix(path, "/")
	if s.BaseURL != "" {
		return strings.TrimSuffix(s.BaseURL, "/") + "/" + p
	}
	return fmt.Sprintf("%s://%s%s", s.scheme(), s.host(), s.objectPath(p))
}

func (s *S3) newRequest(method, path string, body []byte) (*http.Request, error) {
	u := &url.URL{
		Scheme: s.scheme(),
		Host:   s.host(),
		Path:   s.objectPath(path),
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u.String(), rdr)
	if err != nil {
		return nil, err
	}
	s.sign(req, body)
	return req, nil
}

func (s *S3) sign(req *http.Request, body []byte) {
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if req.Header.Get("Host") == "" {
		req.Header.Set("Host", req.URL.Host)
	}

	signedHeaders, canonicalHeaders := canonicalHeaderString(req)
	canonical := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.Query().Encode(),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := dateStamp + "/" + s.region() + "/s3/aws4_request"
	sts := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonical)),
	}, "\n")

	signingKey := s3SigningKey(s.SecretKey, dateStamp, s.region(), "s3")
	sig := hex.EncodeToString(hmacSHA256(signingKey, []byte(sts)))
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.AccessKey, scope, signedHeaders, sig,
	))
}

func canonicalHeaderString(req *http.Request) (signed, canonical string) {
	keys := make([]string, 0, len(req.Header)+1)
	seen := map[string]bool{}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if seen[lk] {
			continue
		}
		seen[lk] = true
		keys = append(keys, lk)
	}
	if !seen["host"] {
		keys = append(keys, "host")
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		val := strings.TrimSpace(req.Header.Get(k))
		if k == "host" && val == "" {
			val = req.URL.Host
		}
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(val)
		b.WriteByte('\n')
	}
	return strings.Join(keys, ";"), b.String()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func s3SigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}
