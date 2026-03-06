package alipay

import (
	"fmt"
)

// AuthClient handles Alipay Mini Program authentication.
type AuthClient struct {
	client *Client
}

// Login exchanges an auth_code from the Alipay mini program for user info.
func (a *AuthClient) Login(authCode string) (*OAuthToken, error) {
	token, err := a.client.SystemOauthToken(authCode)
	if err != nil {
		return nil, fmt.Errorf("alipay/auth: login failed: %w", err)
	}
	return token, nil
}
