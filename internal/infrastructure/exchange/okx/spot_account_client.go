package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"xarb/internal/domain/service"
)

// SpotAccountClient OKX 现货账户查询客户端
type SpotAccountClient struct {
	*APIClient
}

// NewSpotAccountClient 创建现货账户客户端
func NewSpotAccountClient(client *APIClient) *SpotAccountClient {
	return &SpotAccountClient{APIClient: client}
}

// balanceResponse OKX 账户余额 API 响应结构
type balanceResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Details []struct {
			Ccy       string `json:"ccy"`       // 币种
			CcyId     string `json:"ccyId"`     // 币种 ID
			Balance   string `json:"balance"`   // 账户余额
			FrozenBal string `json:"frozenBal"` // 冻结余额
			AvailBal  string `json:"availBal"`  // 可用余额
		} `json:"details"`
		TotalBal string `json:"totalBal"` // 总余额（USD）
	} `json:"data"`
}

// GetAccount 获取现货账户信息
func (c *SpotAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现详细的账户信息返回
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取现货总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// OKX API: GET /api/v5/account/balance
	resp, err := c.fetchBalance(ctx)
	if err != nil {
		return 0, err
	}

	if resp.Code != "0" {
		return 0, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("no balance data returned")
	}

	// OKX 第一个账户数据
	acctData := resp.Data[0]

	// 尝试直接获取总余额（如果以 USD 计价）
	if acctData.TotalBal != "" {
		totalBal, err := strconv.ParseFloat(acctData.TotalBal, 64)
		if err == nil && totalBal > 0 {
			return totalBal, nil
		}
	}

	// 否则手动计算各币种的 USD 价值
	priceCache := make(map[string]float64)
	var totalUSDT float64

	for _, detail := range acctData.Details {
		balance, err := strconv.ParseFloat(detail.AvailBal, 64)
		if err != nil || balance <= 0 {
			continue
		}

		// USDT 和 USDC 直接计入
		ccy := strings.ToUpper(strings.TrimSpace(detail.Ccy))
		if ccy == "USDT" || ccy == "USDC" {
			totalUSDT += balance
			continue
		}

		// 其他币种转换为 USDT
		value, err := c.assetToUSDT(ctx, detail.Ccy, balance, priceCache)
		if err != nil {
			// 忽略无法转换的资产
			continue
		}
		totalUSDT += value
	}

	return totalUSDT, nil
}

// fetchBalance 调用 OKX 账户余额接口
func (c *SpotAccountClient) fetchBalance(ctx context.Context) (*balanceResponse, error) {
	// OKX API: GET /api/v5/account/balance
	path := "/api/v5/account/balance"

	body, err := c.signedQueryRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch okx spot balance: %w", err)
	}

	var resp balanceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal okx balance response: %w", err)
	}

	return &resp, nil
}

// assetToUSDT 将资产转换为 USDT 价值
func (c *SpotAccountClient) assetToUSDT(ctx context.Context, asset string, amount float64, cache map[string]float64) (float64, error) {
	if amount <= 0 {
		return 0, nil
	}

	asset = strings.ToUpper(strings.TrimSpace(asset))

	// USDT 和 USDC 直接返回
	if asset == "USDT" || asset == "USDC" {
		return amount, nil
	}

	// 检查缓存
	if price, ok := cache[asset]; ok {
		return amount * price, nil
	}

	// TODO: 从价格源获取价格（需要集成价格服务）
	// 暂时返回 0，避免阻塞
	return 0, fmt.Errorf("no price available for %s", asset)
}
