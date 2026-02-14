package binance

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// signedRequest is shared helper for signed REST calls.
func (c *APIClient) signedRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", "5000")
	}

	query := params.Encode()
	signature := c.credentials.Sign(query)
	endpoint := fmt.Sprintf("%s%s?%s&signature=%s", strings.TrimRight(c.baseURL, "/"), path, query, signature)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", c.credentials.APIKey())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return nil, fmt.Errorf("account http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
