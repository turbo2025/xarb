package bybit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FundingRateClient Bybit 资金费率 REST 客户端
type FundingRateClient struct {
	baseURL string
	client  *http.Client
}

// BybitFundingRateResp Bybit 资金费率响应
type BybitFundingRateResp struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol      string `json:"symbol"`
			FundingRate string `json:"fundingRate"`
			FundingTime string `json:"fundingTime"`
		} `json:"list"`
	} `json:"result"`
}

// NewFundingRateClient 创建 Bybit REST 客户端
func NewFundingRateClient(baseURL string) *FundingRateClient {
	if baseURL == "" {
		baseURL = "https://api.bybit.com"
	}
	return &FundingRateClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetFundingRate 获取单个合约的资金费率
func (c *FundingRateClient) GetFundingRate(symbol string) (string, error) {
	// Bybit 使用 v5 API
	url := fmt.Sprintf("%s/v5/market/funding/history?category=linear&symbol=%s&limit=1", c.baseURL, symbol)
	resp, err := c.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("bybit api error: %d %s", resp.StatusCode, string(body))
	}

	var result BybitFundingRateResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.RetCode != 0 {
		return "", fmt.Errorf("bybit api error: %s", result.RetMsg)
	}

	if len(result.Result.List) > 0 {
		return result.Result.List[0].FundingRate, nil
	}
	return "0", nil
}

// GetAllFundingRates 批量获取所有永续合约资金费率
func (c *FundingRateClient) GetAllFundingRates() (map[string]string, error) {
	// Bybit 没有一个端点返回所有资金费率，需要逐个查询
	// 这里返回空实现，调用者需要传入符号列表
	return map[string]string{}, nil
}

// GetFundingRateForSymbols 获取指定符号列表的资金费率
func (c *FundingRateClient) GetFundingRateForSymbols(symbols []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, symbol := range symbols {
		rate, err := c.GetFundingRate(symbol)
		if err != nil {
			continue // 继续查询其他
		}
		result[symbol] = rate
	}
	return result, nil
}
