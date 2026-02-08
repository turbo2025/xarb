package bybit

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// SpotOrderClient Bybit 现货订单客户端
type SpotOrderClient struct {
	*clientFields
	baseURL string
}

// NewSpotOrderClient 创建 Bybit 现货订单客户端
func NewSpotOrderClient(apiKey, apiSecret string) *SpotOrderClient {
	return &SpotOrderClient{
		clientFields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: &http.Client{Timeout: 10 * time.Second},
		},
		baseURL: "https://api.bybit.com",
	}
}

// PlaceOrder 下单
// symbol: BTCUSDT
// side: Buy/Sell
// quantity: 0.01
// price: 50000.0
// isMarket: true 表示市价单，false 表示限价单
func (c *SpotOrderClient) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error) {
	// TODO: 实现 POST /v5/order/create
	// Bybit 现货 API 文档: https://bybit-exchange.cn/zh-CN/help-center/article/SPOT_ORDER
	return "", fmt.Errorf("not implemented")
}

// CancelOrder 取消订单
func (c *SpotOrderClient) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	// TODO: 实现 POST /v5/order/cancel
	return fmt.Errorf("not implemented")
}

// GetOrderStatus 查询订单状态
func (c *SpotOrderClient) GetOrderStatus(ctx context.Context, symbol string, orderId string) (*OrderStatus, error) {
	// TODO: 实现 GET /v5/order/realtime?category=spot
	return nil, fmt.Errorf("not implemented")
}

// GetFundingRate 获取资金费率（现货不适用，但保持接口一致）
func (c *SpotOrderClient) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	// 现货交易不适用资金费率
	return 0, nil
}
