package playerauctions

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const BaseURL = "https://seller-api.playerauctions.com"

type Client struct {
	baseURL string
	apiKey  string
	secret  string
	http    *http.Client
	now     func() time.Time
}

func NewClient(apiKey, secret string) (*Client, error) {
	if apiKey == "" || secret == "" {
		return nil, fmt.Errorf("set PLAYERAUCTIONS_API_KEY and PLAYERAUCTIONS_API_SECRET")
	}
	return &Client{
		baseURL: BaseURL, apiKey: apiKey, secret: secret, now: time.Now,
		http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

type APIError struct {
	StatusCode int
	Code       int
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("PlayerAuctions HTTP %d, code %d: %s (requestId=%s)", e.StatusCode, e.Code, e.Message, e.RequestID)
}

func signature(secret, apiKey, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(apiKey + timestamp))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Sign exactly the bytes sent. GET requests use an empty body. Creation requests
// are never automatically retried: a timeout may hide a successful creation.
func (c *Client) do(ctx context.Context, method, path string, input, output any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	req.Header.Set("X-PA-API-KEY", c.apiKey)
	req.Header.Set("X-PA-TIMESTAMP", timestamp)
	req.Header.Set("X-PA-SIGN", signature(c.secret, c.apiKey, timestamp, body))
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PlayerAuctions %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	var envelope struct {
		Code      int             `json:"code"`
		Message   string          `json:"message"`
		RequestID string          `json:"requestId"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&envelope); err != nil {
		return fmt.Errorf("PlayerAuctions %s: HTTP %d, invalid JSON response: %w", path, resp.StatusCode, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code != 10000 {
		return &APIError{resp.StatusCode, envelope.Code, envelope.Message, envelope.RequestID}
	}
	if len(envelope.Data) == 0 || bytes.Equal(envelope.Data, []byte("null")) {
		return fmt.Errorf("PlayerAuctions %s: missing response data (requestId=%s)", path, envelope.RequestID)
	}
	if err := json.Unmarshal(envelope.Data, output); err != nil {
		return fmt.Errorf("decode PlayerAuctions %s data: %w", path, err)
	}
	return nil
}
