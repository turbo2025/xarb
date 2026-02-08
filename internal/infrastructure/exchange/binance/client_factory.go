package binance

import (
	"net/http"
	"time"
)

// clientFields 共享客户端字段
type clientFields struct {
	apiKey    string
	apiSecret string
	client    *http.Client
	baseURL   string
}

// ===== Manager 结构 =====

// FuturesManager Binance 期货统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

// NewFuturesManager 创建 Binance 期货管理器
func NewFuturesManager(apiKey, apiSecret, baseURL string) *FuturesManager {
	client := NewFuturesClient(apiKey, apiSecret, baseURL)
	return &FuturesManager{
		Order:    client.OrderClient(),
		Account:  client.AccountClient(),
		Position: client.PositionClient(),
	}
}

// SpotManager Binance 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Binance 现货管理器
func NewSpotManager(apiKey, apiSecret, baseURL string) *SpotManager {
	client := NewSpotClient(apiKey, apiSecret, baseURL)
	return &SpotManager{
		Order:    client.OrderClient(),
		Account:  client.AccountClient(),
		Position: client.PositionClient(),
	}
}

// ===== 内部工厂函数 =====

// FuturesClient Binance 期货统一客户端
type FuturesClient struct {
	fields *clientFields
}

// NewFuturesClient 创建 Binance 期货客户端工厂
func NewFuturesClient(apiKey, apiSecret, baseURL string) *FuturesClient {
	return &FuturesClient{
		fields: &clientFields{
			apiKey:    apiKey,
			apiSecret: apiSecret,
			client:    &http.Client{Timeout: 10 * time.Second},
			baseURL:   baseURL,
		},
	}
}

// AccountClient 返回账户查询客户端
func (f *FuturesClient) AccountClient() *FuturesAccountClient {
	return &FuturesAccountClient{f.fields}
}

// OrderClient 返回订单客户端
func (f *FuturesClient) OrderClient() *FuturesOrderClient {
	return &FuturesOrderClient{
		clientFields: f.fields,
		baseURL:      f.fields.baseURL,
	}
}

// PositionClient 返回持仓查询客户端
func (f *FuturesClient) PositionClient() *FuturesPositionClient {
	return &FuturesPositionClient{f.fields}
}

// SpotClient Binance 现货统一客户端
type SpotClient struct {
	fields *clientFields
}

// NewSpotClient 创建 Binance 现货客户端工厂
func NewSpotClient(apiKey, apiSecret, baseURL string) *SpotClient {
	return &SpotClient{
		fields: &clientFields{
			apiKey:    apiKey,
			apiSecret: apiSecret,
			client:    &http.Client{Timeout: 10 * time.Second},
			baseURL:   baseURL,
		},
	}
}

// AccountClient 返回账户查询客户端
func (s *SpotClient) AccountClient() *SpotAccountClient {
	return &SpotAccountClient{s.fields}
}

// OrderClient 返回订单客户端
func (s *SpotClient) OrderClient() *SpotOrderClient {
	return &SpotOrderClient{
		clientFields: s.fields,
		baseURL:      s.fields.baseURL,
	}
}

// PositionClient 返回持仓查询客户端
func (s *SpotClient) PositionClient() *SpotPositionClient {
	return &SpotPositionClient{s.fields}
}
