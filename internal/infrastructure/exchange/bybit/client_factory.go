package bybit

import (
	"net/http"
)

// clientFields 共享客户端字段
type clientFields struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	baseURL    string
}

// ===== Manager 结构 =====

// LinearManager Bybit 线性期货（perpetual）统一管理器
type LinearManager struct {
	Order    *LinearOrderClient
	Account  *LinearAccountClient
	Position *LinearPositionClient
}

// NewLinearManager 创建 Bybit 线性期货管理器
func NewLinearManager(apiKey, apiSecret, baseURL string, httpClient *http.Client) *LinearManager {
	client := NewLinearClient(apiKey, apiSecret, baseURL, httpClient)
	return &LinearManager{
		Order:    client.OrderClient(),
		Account:  client.AccountClient(),
		Position: client.PositionClient(),
	}
}

// SpotManager Bybit 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Bybit 现货管理器
func NewSpotManager(apiKey, apiSecret, baseURL string, httpClient *http.Client) *SpotManager {
	client := NewSpotClient(apiKey, apiSecret, baseURL, httpClient)
	return &SpotManager{
		Order:    client.OrderClient(),
		Account:  client.AccountClient(),
		Position: client.PositionClient(),
	}
}

// ===== 内部工厂函数 =====

// LinearClient Bybit 线性期货（perpetual）统一客户端
type LinearClient struct {
	fields *clientFields
}

// NewLinearClient 创建 Bybit 线性期货客户端工厂
func NewLinearClient(apiKey, apiSecret, baseURL string, httpClient *http.Client) *LinearClient {
	return &LinearClient{
		fields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: httpClient,
			baseURL:    baseURL,
		},
	}
}

// AccountClient 返回账户查询客户端
func (l *LinearClient) AccountClient() *LinearAccountClient {
	return &LinearAccountClient{l.fields}
}

// OrderClient 返回订单客户端
func (l *LinearClient) OrderClient() *LinearOrderClient {
	return &LinearOrderClient{
		clientFields: l.fields,
		baseURL:      l.fields.baseURL,
	}
}

// PositionClient 返回持仓查询客户端
func (l *LinearClient) PositionClient() *LinearPositionClient {
	return &LinearPositionClient{l.fields}
}

// SpotClient Bybit 现货统一客户端
type SpotClient struct {
	fields *clientFields
}

// NewSpotClient 创建 Bybit 现货客户端工厂
func NewSpotClient(apiKey, apiSecret, baseURL string, httpClient *http.Client) *SpotClient {
	return &SpotClient{
		fields: &clientFields{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			httpClient: httpClient,
			baseURL:    baseURL,
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
