package alipay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds the Alipay Mini Program configuration.
type Config struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	Gateway    string // optional; default https://openapi.alipay.com/gateway.do
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

func (c *Client) gateway() string {
	if c.config.Gateway != "" {
		return c.config.Gateway
	}
	return "https://openapi.alipay.com/gateway.do"
}

func (c *Client) doSigned(method string, biz map[string]string, result any) error {
	params := map[string]string{
		"app_id":    c.config.AppID,
		"method":    method,
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"version":   "1.0",
	}
	for k, v := range biz {
		params[k] = v
	}
	sign, err := c.signParams(params)
	if err != nil {
		return err
	}
	params["sign"] = sign

	resp, err := c.httpClient.Post(c.gateway(), "application/x-www-form-urlencoded", strings.NewReader(formValues(params).Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("alipay: failed to read response body: %w", err)
	}
	if c.config.PublicKey != "" {
		if err := verifyResponseSign(c.config.PublicKey, body); err != nil {
			return fmt.Errorf("alipay: verify sign: %w", err)
		}
	}
	return json.Unmarshal(body, result)
}

func verifyResponseSign(publicKey string, body []byte) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	signRaw, ok := envelope["sign"]
	if !ok {
		return nil
	}
	var sign string
	if err := json.Unmarshal(signRaw, &sign); err != nil || sign == "" {
		return nil
	}
	var content json.RawMessage
	for k, v := range envelope {
		if k == "sign" || k == "sign_type" {
			continue
		}
		content = v
		break
	}
	if len(content) == 0 {
		return nil
	}
	return verifyAlipaySign(publicKey, string(content), sign)
}

// SystemOauthToken exchanges an auth_code for access token using RSA2-signed POST.
func (c *Client) SystemOauthToken(authCode string) (*OAuthToken, error) {
	var result struct {
		Response OAuthToken `json:"alipay_system_oauth_token_response"`
		ErrResp  struct {
			Code    string `json:"code"`
			Msg     string `json:"msg"`
			SubCode string `json:"sub_code"`
			SubMsg  string `json:"sub_msg"`
		} `json:"error_response"`
	}

	err := c.doSigned("alipay.system.oauth.token", map[string]string{
		"grant_type": "authorization_code",
		"code":       authCode,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("alipay: oauth token request failed: %w", err)
	}
	if result.ErrResp.Code != "" && result.ErrResp.Code != "10000" {
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
	Code         string `json:"code"`
	Msg          string `json:"msg"`
}
