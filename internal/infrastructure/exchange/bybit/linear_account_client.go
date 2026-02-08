package bybit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"xarb/internal/domain/service"
)

// LinearAccountClient Bybit 永续合约账户查询客户端
type LinearAccountClient struct {
	*clientFields
}

// NewLinearAccountClient 创建 Bybit 永续合约账户客户端
func NewLinearAccountClient(apiKey, apiSecret string) *LinearAccountClient {
	return &LinearAccountClient{
		clientFields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: &http.Client{Timeout: 10 * time.Second},
		},
	}
}

// accountResponse API 响应结构
type accountResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		MID                   string `json:"mid"`
		AccountIMRate         string `json:"accountIMRate"`
		IsLiquidating         bool   `json:"isLiquidating"`
		UID                   string `json:"uid"`
		UnifiedMarginStatus   string `json:"unifiedMarginStatus"`
		MarginMode            string `json:"marginMode"`
		UpdatedTime           string `json:"updatedTime"`
		WalletBalance         string `json:"walletBalance"`
		AccountEquity         string `json:"accountEquity"`
		TotalOrderIM          string `json:"totalOrderIM"`
		TotalPositionMM       string `json:"totalPositionMM"`
		TotalAvailableBalance string `json:"totalAvailableBalance"`
		Coin                  []struct {
			Coin                string `json:"coin"`
			Equity              string `json:"equity"`
			UsdValue            string `json:"usdValue"`
			WalletBalance       string `json:"walletBalance"`
			BorrowAmount        string `json:"borrowAmount"`
			AvailableToBorrow   string `json:"availableToBorrow"`
			AvailableToWithdraw string `json:"availableToWithdraw"`
			AccruedInterest     string `json:"accruedInterest"`
			TotalOrderIM        string `json:"totalOrderIM"`
			TotalPositionMM     string `json:"totalPositionMM"`
			MMRate              string `json:"mmRate"`
			IMRate              string `json:"imRate"`
			RiskRate            string `json:"riskRate"`
		} `json:"coin"`
	} `json:"result"`
}

// positionResponse 持仓响应
type positionResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			PositionIdx     int    `json:"positionIdx"`
			TradeMode       int    `json:"tradeMode"`
			RiskID          int    `json:"riskId"`
			RiskLimitValue  string `json:"riskLimitValue"`
			Symbol          string `json:"symbol"`
			Side            string `json:"side"`
			Size            string `json:"size"`
			PositionValue   string `json:"positionValue"`
			EntryPrice      string `json:"entryPrice"`
			Leverage        string `json:"leverage"`
			PosLeverage     string `json:"posLeverage"`
			MarkPrice       string `json:"markPrice"`
			LiqPrice        string `json:"liqPrice"`
			BustPrice       string `json:"bustPrice"`
			IM              string `json:"im"`
			MM              string `json:"mm"`
			RealisedPnl     string `json:"realisedPnl"`
			UnrealisedPnl   string `json:"unrealisedPnl"`
			CumRealisedPnl  string `json:"cumRealisedPnl"`
			SessionAvgPrice string `json:"sessionAvgPrice"`
			Opm             string `json:"opm"`
			Tp              string `json:"tp"`
			Sl              string `json:"sl"`
			TpslMode        string `json:"tpslMode"`
			CreatedTime     string `json:"createdTime"`
			UpdatedTime     string `json:"updatedTime"`
			IsUsed          bool   `json:"isUsed"`
		} `json:"list"`
	} `json:"result"`
}

// openOrderResponse 挂单响应
type openOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID      string `json:"orderId"`
			OrderLinkID  string `json:"orderLinkId"`
			Symbol       string `json:"symbol"`
			Price        string `json:"price"`
			Qty          string `json:"qty"`
			Side         string `json:"side"`
			IsLeverage   string `json:"isLeverage"`
			Status       string `json:"status"`
			LeavesQty    string `json:"leavesQty"`
			LeavesValue  string `json:"leavesValue"`
			CumExecQty   string `json:"cumExecQty"`
			CumExecValue string `json:"cumExecValue"`
			AvgPrice     string `json:"avgPrice"`
			CreatedTime  string `json:"createdTime"`
			UpdatedTime  string `json:"updatedTime"`
			CancelType   string `json:"cancelType"`
		} `json:"list"`
	} `json:"result"`
}

// orderHistoryResponse 订单历史响应
type orderHistoryResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID         string `json:"orderId"`
			OrderLinkID     string `json:"orderLinkId"`
			Symbol          string `json:"symbol"`
			Price           string `json:"price"`
			Qty             string `json:"qty"`
			Side            string `json:"side"`
			Status          string `json:"status"`
			CumExecQty      string `json:"cumExecQty"`
			CumExecValue    string `json:"cumExecValue"`
			AvgPrice        string `json:"avgPrice"`
			Commission      string `json:"commission"`
			CommissionAsset string `json:"commissionAsset"`
			CreatedTime     string `json:"createdTime"`
			UpdatedTime     string `json:"updatedTime"`
			RealisedProfit  string `json:"realisedProfit"`
		} `json:"list"`
	} `json:"result"`
}

// GetAccount 获取账户信息
func (c *LinearAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现 GET /v5/account/wallet-balance
	return nil, fmt.Errorf("not implemented")
}

// GetPositions 获取持仓
func (c *LinearAccountClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/position/list
	return nil, fmt.Errorf("not implemented")
}

// GetOpenOrders 获取挂单
func (c *LinearAccountClient) GetOpenOrders(ctx context.Context, symbol string) ([]*service.OpenOrderInfo, error) {
	// TODO: 实现 GET /v5/order/realtime
	return nil, fmt.Errorf("not implemented")
}

// GetOrderHistory 获取订单历史
func (c *LinearAccountClient) GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*service.OrderLog, error) {
	// TODO: 实现 GET /v5/order/history
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取余额
func (c *LinearAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// TODO: 实现 GET /v5/account/wallet-balance 并提取总余额
	return 0, fmt.Errorf("not implemented")
}
