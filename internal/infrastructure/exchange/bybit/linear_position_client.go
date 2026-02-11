package bybit

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// FuturesPositionClient Bybit 期货持仓客户端
type FuturesPositionClient struct {
	*ClientFields
}

// GetPositions 获取永续合约持仓
func (c *FuturesPositionClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/position/list?category=linear
	// Bybit 永续合约持仓 API: https://bybit-exchange.cn/zh-CN/help-center/article/POSITION_API
	return nil, fmt.Errorf("not implemented")
}

// GetPosition 获取单个交易对的持仓
func (c *FuturesPositionClient) GetPosition(ctx context.Context, symbol string) (*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/position/list?category=linear&symbol=BTCUSDT
	return nil, fmt.Errorf("not implemented")
}

// ClosePosition 平仓（发送反向订单）
func (c *FuturesPositionClient) ClosePosition(ctx context.Context, symbol string) error {
	// TODO: 实现平仓逻辑
	return fmt.Errorf("not implemented")
}

// GetLiquidationPrice 获取清算价格
func (c *FuturesPositionClient) GetLiquidationPrice(ctx context.Context, symbol string) (float64, error) {
	// TODO: 从持仓数据计算清算价格
	return 0, fmt.Errorf("not implemented")
}
