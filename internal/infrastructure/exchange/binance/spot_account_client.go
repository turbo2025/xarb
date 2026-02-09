package binance

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"xarb/internal/domain/service"
)

// SpotAccountClient Binance 现货账户查询客户端
type SpotAccountClient struct {
	credentials *Credentials
	httpClient  *http.Client
	baseURL     string
}

// NewSpotAccountClient 创建 Binance 现货账户客户端
func NewSpotAccountClient(apiKey, apiSecret string) *SpotAccountClient {
	return &SpotAccountClient{
		credentials: NewCredentials(apiKey, apiSecret),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		baseURL:     "https://api.binance.com",
	}
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

// GetPositions 获取现货持仓（实际是代币余额）
func (c *SpotAccountClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /api/v3/account 并返回非零余额作为 "持仓"
	// 现货没有真正的持仓概念，只有钱包余额
	return nil, fmt.Errorf("not implemented")
}

// GetOpenOrders 获取现货挂单
func (c *SpotAccountClient) GetOpenOrders(ctx context.Context, symbol string) ([]*service.OpenOrderInfo, error) {
	// TODO: 实现 GET /api/v3/openOrders?symbol=BTCUSDT
	return nil, fmt.Errorf("not implemented")
}

// GetOrderHistory 获取现货订单历史
func (c *SpotAccountClient) GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*service.OrderLog, error) {
	// TODO: 实现 GET /api/v3/allOrders?symbol=BTCUSDT&limit=100
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取现货总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// TODO: 实现 GET /api/v3/account 并计算总 USDT 价值
	return 0, fmt.Errorf("not implemented")
}
