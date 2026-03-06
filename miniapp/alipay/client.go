package alipay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds the Alipay Mini Program configuration.
type Config struct {
	AppID      string
	PrivateKey string
	PublicKey  string
}

// Client is the Alipay Mini Program SDK client.
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates an Alipay client with the given configuration.
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

func (c *Client) doGet(url string, result any) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("alipay: failed to read response body: %w", err)
	}
	return json.Unmarshal(body, result)
}

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
		return fmt.Errorf("alipay: failed to read response body: %w", err)
	}
	return json.Unmarshal(body, result)
}

// SystemOauthToken exchanges an auth_code for access token.
func (c *Client) SystemOauthToken(authCode string) (*OAuthToken, error) {
	url := fmt.Sprintf(
		"https://openapi.alipay.com/gateway.do?method=alipay.system.oauth.token&app_id=%s&grant_type=authorization_code&code=%s",
		c.config.AppID, authCode,
	)

	var result struct {
		Response OAuthToken `json:"alipay_system_oauth_token_response"`
		ErrResp  struct {
			Code string `json:"code"`
			Msg  string `json:"msg"`
		} `json:"error_response"`
	}

	if err := c.doGet(url, &result); err != nil {
		return nil, fmt.Errorf("alipay: oauth token request failed: %w", err)
	}
	if result.ErrResp.Code != "" {
		return nil, fmt.Errorf("alipay: oauth error %s: %s", result.ErrResp.Code, result.ErrResp.Msg)
	}

	return &result.Response, nil
}

// OAuthToken holds the OAuth token response from Alipay.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	AlipayUserID string `json:"alipay_user_id"`
	UserID       string `json:"user_id"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}
