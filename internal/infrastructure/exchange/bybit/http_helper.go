package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

	return c.doSignedRequest(req, string(body))
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

	return c.doSignedRequest(req, query)
}

func (c *APIClient) doSignedRequest(req *http.Request, payload string) ([]byte, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recvWindow := "5000"

	// Bybit V5 signature: timestamp + apiKey + recvWindow + payload
	signStr := timestamp + c.credentials.APIKey() + recvWindow + payload
	signature := c.credentials.Sign(signStr)

	req.Header.Set("X-BAPI-API-KEY", c.credentials.APIKey())
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Set("X-BAPI-SIGN", signature)

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
		return nil, fmt.Errorf("bybit http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
