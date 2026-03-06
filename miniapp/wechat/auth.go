package wechat

import (
	"fmt"
)

// AuthClient handles WeChat Mini Program authentication (code2Session).
type AuthClient struct {
	client *Client
}

// Session holds the result of a code2Session call.
type Session struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
}

// Code2Session exchanges a login code for a session, the primary
// authentication flow for WeChat Mini Programs.
func (a *AuthClient) Code2Session(code string) (*Session, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		a.client.config.AppID,
		a.client.config.AppSecret,
		code,
	)

	var result struct {
		Session
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := a.client.doGet(url, &result); err != nil {
		return nil, fmt.Errorf("wechat/auth: code2session request failed: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat/auth: code2session error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return &result.Session, nil
}

// GetPhoneNumber retrieves the user's phone number using the phone code
// (requires the getPhoneNumber button in the mini program).
func (a *AuthClient) GetPhoneNumber(phoneCode string) (*PhoneInfo, error) {
	token, err := a.client.GetAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s",
		token,
	)

	var result struct {
		ErrCode   int       `json:"errcode"`
		ErrMsg    string    `json:"errmsg"`
		PhoneInfo PhoneInfo `json:"phone_info"`
	}

	payload := map[string]string{"code": phoneCode}
	if err := a.client.doPostJSON(url, payload, &result); err != nil {
		return nil, fmt.Errorf("wechat/auth: getPhoneNumber failed: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat/auth: getPhoneNumber error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return &result.PhoneInfo, nil
}

// PhoneInfo holds the user's phone number info from WeChat.
type PhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
}
