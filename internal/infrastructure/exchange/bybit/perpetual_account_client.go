package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"xarb/internal/domain/service"
)

// PerpetualAccountClient Bybit 永续合约账户查询客户端
type PerpetualAccountClient struct {
	*APIClient
}

// NewPerpetualAccountClient 创建永续合约账户客户端
func NewPerpetualAccountClient(client *APIClient) *PerpetualAccountClient {
	return &PerpetualAccountClient{APIClient: client}
}

// PerpetualAccountResponse API 响应结构
type PerpetualAccountResponse struct {
	RetCode int    `json:"retCode"` // 返回码 0表示成功
	RetMsg  string `json:"retMsg"`  // 返回信息
	Result  struct {
		List []struct {
			AccountType            string `json:"accountType"`            // 账户类型（UNIFIED统一账户）
			TotalInitialMargin     string `json:"totalInitialMargin"`     // 初始保证金总额
			TotalMaintenanceMargin string `json:"totalMaintenanceMargin"` // 维持保证金总额
			TotalEquity            string `json:"totalEquity"`            // 账户权益总额
			TotalMarginBalance     string `json:"totalMarginBalance"`     // 总保证金余额（钱包余额+持仓未实现盈亏）
			TotalAvailableBalance  string `json:"totalAvailableBalance"`  // 可用保证金余额
			TotalPerpUPL           string `json:"totalPerpUPL"`           // 永续合约未实现盈亏
			TotalWalletBalance     string `json:"totalWalletBalance"`     // 钱包余额（所有币种换算为USDT）
			AccountIMRate          string `json:"accountIMRate"`          // 账户初始保证金率
			AccountMMRate          string `json:"accountMMRate"`          // 账户维持保证金率
			Coin                   []struct {
				Coin             string `json:"coin"`             // 币种代码
				Equity           string `json:"equity"`           // 币种权益
				UsdValue         string `json:"usdValue"`         // 币种USD价值
				WalletBalance    string `json:"walletBalance"`    // 币种钱包余额
				TotalOrderIM     string `json:"totalOrderIM"`     // 币种挂单初始保证金
				TotalPositionMM  string `json:"totalPositionMM"`  // 币种持仓维持保证金
				TotalPositionIM  string `json:"totalPositionIM"`  // 币种持仓初始保证金
				UnrealisedPnl    string `json:"unrealisedPnl"`    // 币种未实现盈亏
				CumRealisedPnl   string `json:"cumRealisedPnl"`   // 币种累计已实现盈亏
				BorrowAmount     string `json:"borrowAmount"`     // 币种借用金额
				MarginCollateral bool   `json:"marginCollateral"` // 是否作为保证金抵押品
				CollateralSwitch bool   `json:"collateralSwitch"` // 抵押品开关状态
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
}

// GetAccount 获取账户信息
func (c *PerpetualAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	params := url.Values{}
	params.Set("accountType", "UNIFIED") // 统一账户类型
	body, err := c.signedQueryRequest(ctx, "GET", "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get bybit futures account: %w", err)
	}

	// 打印原始响应
	// log.Info().Str("response", string(body)).Msg("Bybit wallet-balance raw response")

	var resp PerpetualAccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bybit futures account: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	// 找到 accountType="UNIFIED" 的账户信息
	var unifiedAccountIdx int = -1
	for i := range resp.Result.List {
		if resp.Result.List[i].AccountType == "UNIFIED" {
			unifiedAccountIdx = i
			break
		}
	}

	if unifiedAccountIdx == -1 {
		return nil, fmt.Errorf("no unified account found in response")
	}

	acct := resp.Result.List[unifiedAccountIdx]

	// 解析关键数值
	totalAvailableBalance, _ := strconv.ParseFloat(acct.TotalAvailableBalance, 64)
	totalMarginBalance, _ := strconv.ParseFloat(acct.TotalMarginBalance, 64)
	totalPerpUPL, _ := strconv.ParseFloat(acct.TotalPerpUPL, 64)
	totalWalletBalance, _ := strconv.ParseFloat(acct.TotalWalletBalance, 64)

	// 计算已用保证金 = 总保证金 - 可用保证金
	usedMargin := totalMarginBalance - totalAvailableBalance
	log.Info().
		Float64("totalWalletBalance", totalWalletBalance).
		Float64("totalMarginBalance", totalMarginBalance).
		Float64("totalAvailableBalance", totalAvailableBalance).
		Float64("totalPerpUPL", totalPerpUPL).
		Float64("usedMargin", usedMargin).
		Msg("Bybit account summary")

	return &service.AccountInfo{
		TotalMargin: totalWalletBalance,
		UsedMargin:  usedMargin,
		AvailMargin: totalAvailableBalance,
		UpdatedAt:   time.Now(),
	}, nil
}

// GetBalance 获取余额
func (c *PerpetualAccountClient) GetBalance(ctx context.Context) (float64, error) {
	params := url.Values{}
	params.Set("accountType", "UNIFIED") // 统一账户类型
	body, err := c.signedQueryRequest(ctx, "GET", "/v5/account/wallet-balance", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get bybit futures balance: %w", err)
	}

	log.Info().Str("response", string(body)).Msg("Bybit GetBalance raw response")

	var resp PerpetualAccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal bybit futures balance: %w", err)
	}

	if resp.RetCode != 0 {
		return 0, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	// 找到 accountType="UNIFIED" 的账户信息
	var unifiedAccountIdx int = -1
	for i := range resp.Result.List {
		if resp.Result.List[i].AccountType == "UNIFIED" {
			unifiedAccountIdx = i
			break
		}
	}

	if unifiedAccountIdx == -1 {
		return 0, fmt.Errorf("no unified account found in response")
	}

	acct := resp.Result.List[unifiedAccountIdx]
	totalWalletBalance, _ := strconv.ParseFloat(acct.TotalWalletBalance, 64)
	log.Info().
		Float64("totalWalletBalance", totalWalletBalance).
		Str("totalAvailableBalance", acct.TotalAvailableBalance).
		Str("totalMarginBalance", acct.TotalMarginBalance).
		Str("totalPerpUPL", acct.TotalPerpUPL).
		Msg("Bybit balance info")
	return totalWalletBalance, nil
}
