package binance

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// FuturesAccountClient Binance 期货账户查询客户端
type FuturesAccountClient struct {
	*APIClient
}

// NewFuturesAccountClient 创建期货账户客户端
func NewFuturesAccountClient(client *APIClient) *FuturesAccountClient {
	return &FuturesAccountClient{APIClient: client}
}

// accountResponse API 响应结构
type accountResponse struct {
	FeeTier               int    `json:"feeTier"`
	CanTrade              bool   `json:"canTrade"`
	CanDeposit            bool   `json:"canDeposit"`
	CanWithdraw           bool   `json:"canWithdraw"`
	UpdateTime            int64  `json:"updateTime"`
	TotalInitialMargin    string `json:"totalInitialMargin"`
	TotalMaintMargin      string `json:"totalMaintMargin"`
	TotalWalletBalance    string `json:"totalWalletBalance"`
	TotalUnrealizedProfit string `json:"totalUnrealizedProfit"`
	Positions             []struct {
		Symbol                 string `json:"symbol"`
		InitialMargin          string `json:"initialMargin"`
		MaintMargin            string `json:"maintMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		PositionAmt            string `json:"positionAmt"`
		MaxNotional            string `json:"maxNotional"`
		MarkPrice              string `json:"markPrice"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		ContractType           string `json:"contractType"`
		LeverageBracket        int    `json:"leverageBracket"`
		Isolated               bool   `json:"isolated"`
		PositionSide           string `json:"positionSide"`
		Notional               string `json:"notional"`
		BidNotional            string `json:"bidNotional"`
		AskNotional            string `json:"askNotional"`
		UpdateTime             int64  `json:"updateTime"`
		Leverage               string `json:"leverage"`
	} `json:"positions"`
	UserAssets []struct {
		Asset                  string `json:"asset"`
		WalletBalance          string `json:"walletBalance"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		MarginBalance          string `json:"marginBalance"`
		MaintMargin            string `json:"maintMargin"`
		InitialMargin          string `json:"initialMargin"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		CrossWalletBalance     string `json:"crossWalletBalance"`
		CrossUnPnl             string `json:"crossUnPnl"`
		AvailableBalance       string `json:"availableBalance"`
		MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
		MarginAvailable        string `json:"marginAvailable"`
		UpdateTime             int64  `json:"updateTime"`
	} `json:"userAssets"`
}

// GetAccount 获取期货账户信息
func (c *FuturesAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现 GET /fapi/v2/account 并提取 positions 字段
	// Binance 期货账户 API: https://binance-docs.github.io/apidocs/futures/cn/#user_data-8
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取余额
func (c *FuturesAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// TODO: 实现 GET /fapi/v2/account 并提取总 USDT 价值
	return 0, fmt.Errorf("not implemented")
}
