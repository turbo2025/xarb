package bybit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"xarb/internal/domain/service"
)

// SpotPositionClient Bybit 现货持仓（钱包）客户端
type SpotPositionClient struct {
	*clientFields
}

// NewSpotPositionClient 创建 Bybit 现货持仓客户端
func NewSpotPositionClient(apiKey, apiSecret string) *SpotPositionClient {
	return &SpotPositionClient{
		clientFields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: &http.Client{Timeout: 10 * time.Second},
		},
	}
}

// GetBalances 获取所有币种余额（钱包）
func (c *SpotPositionClient) GetBalances(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT 并返回非零余额作为 "持仓"
	// 现货的 "持仓" 实际上是钱包余额
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取单个币种余额
func (c *SpotPositionClient) GetBalance(ctx context.Context, coin string) (walletBalance float64, availableBalance float64, err error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT 并提取指定 coin 的余额
	return 0, 0, fmt.Errorf("not implemented")
}

// TransferOut 转出（例如转到合约账户）
func (c *SpotPositionClient) TransferOut(ctx context.Context, coin string, amount float64) error {
	// TODO: 实现账户间转账
	// https://bybit-exchange.cn/zh-CN/help-center/article/WALLET_API
	return fmt.Errorf("not implemented")
}

// TransferIn 转入（例如从合约账户转入）
func (c *SpotPositionClient) TransferIn(ctx context.Context, coin string, amount float64) error {
	// TODO: 实现账户间转账
	return fmt.Errorf("not implemented")
}
