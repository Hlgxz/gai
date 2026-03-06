package wechat

import (
	"fmt"
)

// MessageClient handles WeChat Mini Program subscribe messages.
type MessageClient struct {
	client *Client
}

// SubscribeMessage holds the parameters for sending a subscribe message.
type SubscribeMessage struct {
	ToUser     string            `json:"touser"`
	TemplateID string            `json:"template_id"`
	Page       string            `json:"page,omitempty"`
	Data       map[string]MsgVal `json:"data"`
}

// MsgVal is a template message data value.
type MsgVal struct {
	Value string `json:"value"`
}

// SendSubscribe sends a subscribe message to the user.
func (m *MessageClient) SendSubscribe(msg *SubscribeMessage) error {
	token, err := m.client.GetAccessToken()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s",
		token,
	)

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := m.client.doPostJSON(url, msg, &result); err != nil {
		return fmt.Errorf("wechat/message: send failed: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat/message: error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// SendCustomerMessage sends a customer service message.
func (m *MessageClient) SendCustomerMessage(toUser, msgType string, content map[string]any) error {
	token, err := m.client.GetAccessToken()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s",
		token,
	)

	payload := map[string]any{
		"touser":  toUser,
		"msgtype": msgType,
		msgType:   content,
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := m.client.doPostJSON(url, payload, &result); err != nil {
		return fmt.Errorf("wechat/message: customer msg failed: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat/message: error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}
