package bybit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"xarb/internal/domain/service"
)

// LinearPositionClient Bybit 永续合约持仓客户端
type LinearPositionClient struct {
	*clientFields
}

// NewLinearPositionClient 创建 Bybit 永续合约持仓客户端
func NewLinearPositionClient(apiKey, apiSecret string) *LinearPositionClient {
	return &LinearPositionClient{
		clientFields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: &http.Client{Timeout: 10 * time.Second},
		},
	}
}

// GetPositions 获取永续合约持仓
func (c *LinearPositionClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/position/list?category=linear
	// Bybit 永续合约持仓 API: https://bybit-exchange.cn/zh-CN/help-center/article/POSITION_API
	return nil, fmt.Errorf("not implemented")
}

// GetPosition 获取单个交易对的持仓
func (c *LinearPositionClient) GetPosition(ctx context.Context, symbol string) (*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/position/list?category=linear&symbol=BTCUSDT
	return nil, fmt.Errorf("not implemented")
}

// ClosePosition 平仓（发送反向订单）
func (c *LinearPositionClient) ClosePosition(ctx context.Context, symbol string) error {
	// TODO: 实现平仓逻辑
	return fmt.Errorf("not implemented")
}

// GetLiquidationPrice 获取清算价格
func (c *LinearPositionClient) GetLiquidationPrice(ctx context.Context, symbol string) (float64, error) {
	// TODO: 从持仓数据计算清算价格
	return 0, fmt.Errorf("not implemented")
}
