package okx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// signedJSONRequest 发送带 JSON payload 的签名请求
func (c *APIClient) signedJSONRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doSignedRequest(req, path, string(body))
}

// signedQueryRequest 发送带 query 的签名请求
func (c *APIClient) signedQueryRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	var query string
	if params != nil {
		query = params.Encode()
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + path
	if query != "" {
		endpoint += "?" + query
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}

	return c.doSignedRequest(req, path, query)
}

func (c *APIClient) doSignedRequest(req *http.Request, path, payload string) ([]byte, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// OKX signature
	// message = timestamp + method + requestPath + body
	signStr := timestamp + req.Method + path + payload
	signature := c.credentials.Sign(signStr)

	req.Header.Set("OK-ACCESS-KEY", c.credentials.APIKey())
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.credentials.Passphrase())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("okx http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
