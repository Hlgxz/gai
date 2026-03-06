package wechat

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// PayClient handles WeChat payment operations.
type PayClient struct {
	client *Client
}

// Order holds the parameters for creating a unified payment order.
type Order struct {
	Body           string `json:"body"`
	OutTradeNo     string `json:"out_trade_no"`
	TotalFee       int    `json:"total_fee"` // in cents (分)
	SpbillCreateIP string `json:"spbill_create_ip"`
	OpenID         string `json:"openid"`
	TradeType      string `json:"trade_type,omitempty"`
}

// PayResult holds the unified order response used to invoke payment
// from the mini program side.
type PayResult struct {
	PrepayID  string `json:"prepay_id"`
	NonceStr  string `json:"nonce_str"`
	Timestamp string `json:"timestamp"`
	Package   string `json:"package"`
	SignType  string `json:"sign_type"`
	PaySign   string `json:"pay_sign"`
}

// UnifiedOrder creates a WeChat payment order and returns the data needed
// to invoke wx.requestPayment() on the client side.
func (p *PayClient) UnifiedOrder(order *Order) (*PayResult, error) {
	cfg := p.client.config
	if order.TradeType == "" {
		order.TradeType = "JSAPI"
	}

	params := map[string]string{
		"appid":            cfg.AppID,
		"mch_id":           cfg.MchID,
		"nonce_str":        randomNonce(),
		"body":             order.Body,
		"out_trade_no":     order.OutTradeNo,
		"total_fee":        fmt.Sprintf("%d", order.TotalFee),
		"spbill_create_ip": order.SpbillCreateIP,
		"notify_url":       cfg.NotifyURL,
		"trade_type":       order.TradeType,
		"openid":           order.OpenID,
	}

	params["sign"] = signMD5(params, cfg.APIKey)

	xmlBody := mapToXML(params)
	resp, err := http.Post(
		"https://api.mch.weixin.qq.com/pay/unifiedorder",
		"application/xml",
		strings.NewReader(xmlBody),
	)
	if err != nil {
		return nil, fmt.Errorf("wechat/pay: unified order request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := make(map[string]string)
	if err := parseXMLToMap(body, result); err != nil {
		return nil, fmt.Errorf("wechat/pay: failed to parse response: %w", err)
	}

	if result["return_code"] != "SUCCESS" || result["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("wechat/pay: order failed: %s - %s",
			result["return_msg"], result["err_code_des"])
	}

	now := time.Now()
	ts := fmt.Sprintf("%d", now.Unix())
	pkg := "prepay_id=" + result["prepay_id"]

	payParams := map[string]string{
		"appId":     cfg.AppID,
		"timeStamp": ts,
		"nonceStr":  params["nonce_str"],
		"package":   pkg,
		"signType":  "MD5",
	}
	paySign := signMD5(payParams, cfg.APIKey)

	return &PayResult{
		PrepayID:  result["prepay_id"],
		NonceStr:  params["nonce_str"],
		Timestamp: ts,
		Package:   pkg,
		SignType:  "MD5",
		PaySign:   paySign,
	}, nil
}

// VerifyNotify checks the sign of a payment notification callback.
func (p *PayClient) VerifyNotify(body []byte) (map[string]string, error) {
	params := make(map[string]string)
	if err := parseXMLToMap(body, params); err != nil {
		return nil, err
	}

	sign := params["sign"]
	delete(params, "sign")

	expected := signMD5(params, p.client.config.APIKey)
	if sign != expected {
		return nil, fmt.Errorf("wechat/pay: invalid notification signature")
	}

	return params, nil
}

// ---------------------------------------------------------- Helpers

func signMD5(params map[string]string, apiKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(params[k])
	}
	buf.WriteString("&key=")
	buf.WriteString(apiKey)

	h := md5.Sum([]byte(buf.String()))
	return strings.ToUpper(hex.EncodeToString(h[:]))
}

func mapToXML(m map[string]string) string {
	var buf strings.Builder
	buf.WriteString("<xml>")
	for k, v := range m {
		buf.WriteString(fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k))
	}
	buf.WriteString("</xml>")
	return buf.String()
}

func parseXMLToMap(data []byte, m map[string]string) error {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var currentKey string
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("wechat/pay: xml parse error: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			currentKey = t.Name.Local
		case xml.CharData:
			if currentKey != "" && currentKey != "xml" {
				val := strings.TrimSpace(string(t))
				if val != "" {
					m[currentKey] = val
				}
			}
		case xml.EndElement:
			currentKey = ""
		}
	}
	return nil
}

func randomNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("wechat/pay: failed to generate random nonce: %v", err))
	}
	return hex.EncodeToString(b)
}
