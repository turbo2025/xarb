package bybit

import (
	"context"
	"fmt"
)

// SpotPositionClient Bybit 现货持仓（钱包）客户端
type SpotPositionClient struct {
	*APIClient
}

// NewSpotPositionClient 创建现货持仓客户端
func NewSpotPositionClient(client *APIClient) *SpotPositionClient {
	return &SpotPositionClient{APIClient: client}
}

// GetBalances 获取所有币种余额（钱包）
func (c *SpotPositionClient) GetBalances(ctx context.Context) (interface{}, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取单个币种余额
func (c *SpotPositionClient) GetBalance(ctx context.Context, asset string) (free float64, locked float64, err error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT
	return 0, 0, fmt.Errorf("not implemented")
}

// TransferOut 转出（例如转到合约账户）
func (c *SpotPositionClient) TransferOut(ctx context.Context, asset string, amount float64) error {
	// TODO: 实现账户间转账
	return fmt.Errorf("not implemented")
}

// TransferIn 转入（例如从合约账户转入）
func (c *SpotPositionClient) TransferIn(ctx context.Context, asset string, amount float64) error {
	// TODO: 实现账户间转账
	return fmt.Errorf("not implemented")
}
