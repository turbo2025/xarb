package bybit

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

// SpotAccountClient Bybit 现货账户查询客户端
type SpotAccountClient struct {
	*APIClient
}

// NewSpotAccountClient 创建现货账户客户端
func NewSpotAccountClient(client *APIClient) *SpotAccountClient {
	return &SpotAccountClient{APIClient: client}
}

// walletBalanceResponse Bybit wallet-balance API 响应结构
type walletBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			AccountType string `json:"accountType"`
			Coin        []struct {
				Coin                string `json:"coin"`
				Equity              string `json:"equity"`
				UsdValue            string `json:"usdValue"`
				WalletBalance       string `json:"walletBalance"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
				BorrowAmount        string `json:"borrowAmount"`
				AccruedInterest     string `json:"accruedInterest"`
				TotalOrderIM        string `json:"totalOrderIM"`
				TotalPositionIM     string `json:"totalPositionIM"`
				Locked              string `json:"locked"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
}

// GetAccount 获取现货账户信息
func (c *SpotAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT
	// Bybit 现货账户 API: https://bybit-exchange.cn/zh-CN/help-center/article/WALLET_API
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取现货总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	resp, err := c.fetchWalletBalance(ctx, "UNIFIED")
	if err != nil {
		return 0, err
	}

	priceCache := make(map[string]float64)
	var totalUSDT float64

	for _, account := range resp.Result.List {
		for _, coin := range account.Coin {
			// 先尝试使用 usdValue（如果 Bybit 已计算）
			if coin.UsdValue != "" && coin.UsdValue != "0" {
				usdVal, err := strconv.ParseFloat(coin.UsdValue, 64)
				if err == nil && usdVal > 0 {
					totalUSDT += usdVal
					continue
				}
			}

			// 否则手动计算
			balance, err := strconv.ParseFloat(coin.WalletBalance, 64)
			if err != nil || balance <= 0 {
				continue
			}

			value, err := c.assetToUSDT(ctx, coin.Coin, balance, priceCache)
			if err != nil {
				// 忽略无法换算的资产
				continue
			}
			totalUSDT += value
		}
	}

	return totalUSDT, nil
}

// fetchWalletBalance 调用 Bybit wallet-balance 接口
func (c *SpotAccountClient) fetchWalletBalance(ctx context.Context, accountType string) (*walletBalanceResponse, error) {
	params := url.Values{}
	params.Set("accountType", accountType)

	body, err := c.signedQueryRequest(ctx, http.MethodGet, "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, err
	}

	var resp walletBalanceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode wallet-balance response failed: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit wallet-balance error: [%d] %s", resp.RetCode, resp.RetMsg)
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
	endpoint := fmt.Sprintf("%s/v5/market/tickers?category=spot&symbol=%s",
		strings.TrimRight(c.baseURL, "/"), url.QueryEscape(symbol))

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
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				LastPrice string `json:"lastPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode ticker failed: %w", err)
	}

	if data.RetCode != 0 || len(data.Result.List) == 0 {
		return 0, fmt.Errorf("ticker error: [%d] %s", data.RetCode, data.RetMsg)
	}

	price, err := strconv.ParseFloat(data.Result.List[0].LastPrice, 64)
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
