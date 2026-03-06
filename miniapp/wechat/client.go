package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config holds the WeChat Mini Program configuration.
type Config struct {
	AppID     string
	AppSecret string
	MchID     string // merchant ID for payments
	APIKey    string // payment API key
	NotifyURL string // payment callback URL
}

// Client is the unified WeChat Mini Program SDK client.
type Client struct {
	config      Config
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// NewClient creates a WeChat client with the given configuration.
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Auth returns the authentication sub-client.
func (c *Client) Auth() *AuthClient {
	return &AuthClient{client: c}
}

// Pay returns the payment sub-client.
func (c *Client) Pay() *PayClient {
	return &PayClient{client: c}
}

// Message returns the message sub-client.
func (c *Client) Message() *MessageClient {
	return &MessageClient{client: c}
}

// GetAccessToken returns a cached access token, refreshing if expired.
func (c *Client) GetAccessToken() (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.config.AppID, c.config.AppSecret,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("wechat: failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("wechat: failed to decode token response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat: token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return c.accessToken, nil
}

// doGet performs an authenticated GET request.
func (c *Client) doGet(url string, result any) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("wechat: failed to read response body: %w", err)
	}
	return json.Unmarshal(body, result)
}

// doPostJSON performs an authenticated POST request with JSON body.
func (c *Client) doPostJSON(url string, payload any, result any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("wechat: failed to read response body: %w", err)
	}
	return json.Unmarshal(body, result)
}
