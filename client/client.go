package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client with timeout and retries on network / 5xx errors.
type Client struct {
	HTTP    *http.Client
	Retries int
	Backoff time.Duration
}

// New returns a client with a 10s timeout and 2 retries.
func New() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Retries: 2,
		Backoff: 200 * time.Millisecond,
	}
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Do executes req, retrying on transport errors and HTTP 5xx.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	retries := c.Retries
	if retries < 0 {
		retries = 0
	}
	backoff := c.Backoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}

	var last error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff * time.Duration(attempt)):
			}
		}
		resp, err := c.http().Do(req)
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode >= 500 && attempt < retries {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			last = fmt.Errorf("gai/client: status %d", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("gai/client: retries exhausted: %w", last)
}

// GetJSON GETs url and decodes a JSON body into dst.
func (c *Client) GetJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, dst)
}

// PostJSON POSTs JSON payload and decodes the JSON response into dst.
func (c *Client) PostJSON(ctx context.Context, url string, payload, dst any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req, dst)
}

func (c *Client) doJSON(req *http.Request, dst any) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gai/client: status %d: %s", resp.StatusCode, truncate(body, 512))
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
