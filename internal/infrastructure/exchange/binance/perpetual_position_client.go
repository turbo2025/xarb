package binance

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// PerpetualPositionClient Binance perpetual position client
type PerpetualPositionClient struct {
	*APIClient
}

// NewPerpetualPositionClient creates perpetual position client
func NewPerpetualPositionClient(client *APIClient) *PerpetualPositionClient {
	return &PerpetualPositionClient{APIClient: client}
}

// GetPositions gets perpetual positions
func (c *PerpetualPositionClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /fapi/v2/account 并提取 positions 字段
	// Binance 期货持仓 API: https://binance-docs.github.io/apidocs/futures/cn/#user_data-8
	return nil, fmt.Errorf("not implemented")
}

// GetPosition 获取单个交易对的持仓
func (c *PerpetualPositionClient) GetPosition(ctx context.Context, symbol string) (*service.PositionInfo, error) {
	// TODO: 实现 GET /fapi/v2/account 并提取指定 symbol 的持仓
	return nil, fmt.Errorf("not implemented")
}

// ClosePosition 平仓（发送反向订单）
func (c *PerpetualPositionClient) ClosePosition(ctx context.Context, symbol string) error {
	// TODO: 实现平仓逻辑
	return fmt.Errorf("not implemented")
}

// GetLiquidationPrice 获取清算价格
func (c *PerpetualPositionClient) GetLiquidationPrice(ctx context.Context, symbol string) (float64, error) {
	// TODO: 从持仓数据计算清算价格
	return 0, fmt.Errorf("not implemented")
}
