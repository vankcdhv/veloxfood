// Package momo implements the MoMo v2 payment gateway client.
// Switching demo↔prod requires only config changes (endpoint, credentials).
package momo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"project/pkg/config"
)

// Client wraps MoMo v2 API calls (create-payment, IPN verification).
type Client struct {
	cfg        config.MoMoConfig
	httpClient *http.Client
}

// NewClient creates a MoMo client from config.
func NewClient(cfg config.MoMoConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreatePaymentResult holds the MoMo response for a create-payment call.
type CreatePaymentResult struct {
	PayURL string
	// MoMo returns these for tracing/debugging.
	RequestID string
	OrderID   string
}

// CreatePayment builds and signs a MoMo v2 create-payment request, calls the
// gateway, and returns the pay_url the customer must be redirected to.
// Network or gateway errors are returned as typed errors; callers should treat
// them as transient and may retry or mark payment FAILED.
func (c *Client) CreatePayment(ctx context.Context, orderID, orderInfo string, amount int64, redirectURL, ipnURL, requestID string) (*CreatePaymentResult, error) {
	extraData := ""
	requestType := "payWithMethod"

	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
		c.cfg.AccessKey,
		amount,
		extraData,
		ipnURL,
		orderID,
		orderInfo,
		c.cfg.PartnerCode,
		redirectURL,
		requestID,
		requestType,
	)
	signature := c.hmacSHA256(rawSignature)

	body := map[string]any{
		"partnerCode": c.cfg.PartnerCode,
		"accessKey":   c.cfg.AccessKey,
		"requestId":   requestID,
		"amount":      amount,
		"orderId":     orderID,
		"orderInfo":   orderInfo,
		"redirectUrl": redirectURL,
		"ipnUrl":      ipnURL,
		"extraData":   extraData,
		"requestType": requestType,
		"signature":   signature,
		"lang":        "vi",
	}

	respBody, err := c.post(ctx, c.cfg.Endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("momo: create-payment request failed: %w", err)
	}

	var resp struct {
		ResultCode int    `json:"resultCode"`
		Message    string `json:"message"`
		PayURL     string `json:"payUrl"`
		RequestID  string `json:"requestId"`
		OrderID    string `json:"orderId"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("momo: parse create-payment response: %w", err)
	}
	if resp.ResultCode != 0 {
		return nil, fmt.Errorf("momo: create-payment rejected (code=%d): %s", resp.ResultCode, resp.Message)
	}

	slog.InfoContext(ctx, "momo: payment created", "order_id", orderID, "request_id", requestID)
	return &CreatePaymentResult{
		PayURL:    resp.PayURL,
		RequestID: resp.RequestID,
		OrderID:   resp.OrderID,
	}, nil
}

// IPNFields is the subset of MoMo IPN fields used for HMAC-SHA256 verification
// (per MoMo v2 spec — alphabetical order).
type IPNFields struct {
	AccessKey     string
	Amount        int64
	ExtraData     string
	Message       string
	OrderID       string
	OrderInfo     string
	OrderType     string
	PartnerCode   string
	PayType       string
	RequestID     string
	ResponseTime  int64
	ResultCode    int
	TransID       int64
	Signature     string // the signature sent by MoMo
}

// VerifyIPN recomputes the HMAC-SHA256 over MoMo's IPN field set and compares
// it to the signature field. Returns nil if valid, error if tampered/invalid.
func (c *Client) VerifyIPN(fields IPNFields) error {
	raw := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%d&resultCode=%d&transId=%d",
		fields.AccessKey,
		fields.Amount,
		fields.ExtraData,
		fields.Message,
		fields.OrderID,
		fields.OrderInfo,
		fields.OrderType,
		fields.PartnerCode,
		fields.PayType,
		fields.RequestID,
		fields.ResponseTime,
		fields.ResultCode,
		fields.TransID,
	)
	expected := c.hmacSHA256(raw)
	if expected != fields.Signature {
		return fmt.Errorf("momo: IPN signature mismatch")
	}
	return nil
}

// hmacSHA256 computes HMAC-SHA256 of msg using the configured secret key.
func (c *Client) hmacSHA256(msg string) string {
	mac := hmac.New(sha256.New, []byte(c.cfg.SecretKey))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) post(ctx context.Context, url string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return data, nil
}
