package binance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FundingRateClient Binance 资金费率 REST 客户端
type FundingRateClient struct {
	baseURL string
	client  *http.Client
}

// FundingRateResp Binance 资金费率响应
type FundingRateResp struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

// NewFundingRateClient 创建 Binance REST 客户端
func NewFundingRateClient(baseURL string) *FundingRateClient {
	if baseURL == "" {
		baseURL = "https://fapi.binance.com"
	}
	return &FundingRateClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetFundingRate 获取单个合约的资金费率
func (c *FundingRateClient) GetFundingRate(symbol string) (*FundingRateResp, error) {
	url := fmt.Sprintf("%s/fapi/v1/fundingRate?symbol=%s", c.baseURL, symbol)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance api error: %d %s", resp.StatusCode, string(body))
	}

	var result FundingRateResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllFundingRates 批量获取所有永续合约资金费率
func (c *FundingRateClient) GetAllFundingRates() ([]FundingRateResp, error) {
	url := fmt.Sprintf("%s/fapi/v1/fundingRate", c.baseURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance api error: %d %s", resp.StatusCode, string(body))
	}

	var results []FundingRateResp
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetFundingRateHistory 获取资金费率历史
func (c *FundingRateClient) GetFundingRateHistory(symbol string, limit int) ([]FundingRateResp, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	url := fmt.Sprintf("%s/fapi/v1/fundingRate?symbol=%s&limit=%d", c.baseURL, symbol, limit)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance api error: %d %s", resp.StatusCode, string(body))
	}

	var results []FundingRateResp
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}
