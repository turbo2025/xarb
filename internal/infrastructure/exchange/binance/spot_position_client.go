package binance

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// SpotPositionClient Binance 现货持仓（钱包）客户端
type SpotPositionClient struct {
	*APIClient
}

// NewSpotPositionClient 创建现货持仓客户端
func NewSpotPositionClient(client *APIClient) *SpotPositionClient {
	return &SpotPositionClient{APIClient: client}
}

// GetBalances 获取所有币种余额（钱包）
func (c *SpotPositionClient) GetBalances(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /api/v3/account 并返回非零余额作为 "持仓"
	// 现货的 "持仓" 实际上是钱包余额
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取单个币种余额
func (c *SpotPositionClient) GetBalance(ctx context.Context, asset string) (free float64, locked float64, err error) {
	// TODO: 实现 GET /api/v3/account 并提取指定 asset 的余额
	return 0, 0, fmt.Errorf("not implemented")
}

// TransferOut 转出（例如转到合约账户）
func (c *SpotPositionClient) TransferOut(ctx context.Context, asset string, amount float64) error {
	// TODO: 实现账户间转账
	// https://binance-docs.github.io/apidocs/spot/cn/#user_data
	return fmt.Errorf("not implemented")
}

// TransferIn 转入（例如从合约账户转入）
func (c *SpotPositionClient) TransferIn(ctx context.Context, asset string, amount float64) error {
	// TODO: 实现账户间转账
	return fmt.Errorf("not implemented")
}
