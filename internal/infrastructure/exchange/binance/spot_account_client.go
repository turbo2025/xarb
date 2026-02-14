package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"xarb/internal/domain/service"
)

// SpotAccountClient Binance 现货账户查询客户端
type SpotAccountClient struct {
	*APIClient
}

// NewSpotAccountClient 创建现货账户客户端
func NewSpotAccountClient(client *APIClient) *SpotAccountClient {
	return &SpotAccountClient{APIClient: client}
}

// spotAccountResponse 现货账户响应结构
type spotAccountResponse struct {
	MakerCommission  int   `json:"makerCommission"`
	TakerCommission  int   `json:"takerCommission"`
	BuyerCommission  int   `json:"buyerCommission"`
	SellerCommission int   `json:"sellerCommission"`
	CanTrade         bool  `json:"canTrade"`
	CanWithdraw      bool  `json:"canWithdraw"`
	CanDeposit       bool  `json:"canDeposit"`
	UpdateTime       int64 `json:"updateTime"`
	Balances         []struct {
		Asset  string `json:"asset"`
		Free   string `json:"free"`
		Locked string `json:"locked"`
	} `json:"balances"`
}

// GetAccount 获取现货账户信息
func (c *SpotAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现 GET /api/v3/account
	// Binance 现货账户 API: https://binance-docs.github.io/apidocs/spot/cn/#account-information-user_data
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取现货账户总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	resp, err := c.fetchAccount(ctx)
	if err != nil {
		return 0, err
	}

	priceCache := make(map[string]float64)
	var totalUSDT float64

	for _, balance := range resp.Balances {
		free, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return 0, fmt.Errorf("parse free balance for %s: %w", balance.Asset, err)
		}
		locked, err := strconv.ParseFloat(balance.Locked, 64)
		if err != nil {
			return 0, fmt.Errorf("parse locked balance for %s: %w", balance.Asset, err)
		}

		amount := free + locked
		if amount <= 0 {
			continue
		}

		value, err := c.assetToUSDT(ctx, balance.Asset, amount, priceCache)
		if err != nil {
			return 0, err
		}
		totalUSDT += value
	}

	return totalUSDT, nil
}

// fetchAccount 调用 Binance 现货账户接口
func (c *SpotAccountClient) fetchAccount(ctx context.Context) (*spotAccountResponse, error) {
	body, err := c.signedRequest(ctx, http.MethodGet, "/api/v3/account", nil)
	if err != nil {
		return nil, err
	}

	var resp spotAccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode account response failed: %w", err)
	}
	return &resp, nil
}

// assetToUSDT 将任意资产换算为 USDT
func (c *SpotAccountClient) assetToUSDT(ctx context.Context, asset string, amount float64, cache map[string]float64) (float64, error) {
	asset = strings.ToUpper(asset)
	if amount == 0 {
		return 0, nil
	}

	if isUSDStableCoin(asset) {
		return amount, nil
	}

	symbol := asset + "USDT"
	if price, ok := cache[symbol]; ok {
		return amount * price, nil
	}

	price, err := c.fetchTickerPrice(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("get ticker %s failed: %w", symbol, err)
	}
	cache[symbol] = price
	return amount * price, nil
}

// fetchTickerPrice 获取现货 ticker 价格
func (c *SpotAccountClient) fetchTickerPrice(ctx context.Context, symbol string) (float64, error) {
	endpoint := fmt.Sprintf("%s/api/v3/ticker/price?symbol=%s", strings.TrimRight(c.baseURL, "/"), url.QueryEscape(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("ticker http %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode ticker failed: %w", err)
	}

	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse ticker price failed: %w", err)
	}
	return price, nil
}

func isUSDStableCoin(asset string) bool {
	switch asset {
	case "USDT", "USDC", "BUSD", "FDUSD", "TUSD", "DAI", "USDD":
		return true
	default:
		return false
	}
}
