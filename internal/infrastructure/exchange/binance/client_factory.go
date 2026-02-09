package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

// ===== Credentials 凭证 =====

// Credentials 包含 API 凭证和签名方法
type Credentials struct {
	apiKey    string
	apiSecret string
}

// NewCredentials 创建凭证对象
func NewCredentials(apiKey, apiSecret string) *Credentials {
	return &Credentials{
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// Sign 生成 HMAC-SHA256 签名
func (c *Credentials) Sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// APIKey 返回 API Key
func (c *Credentials) APIKey() string {
	return c.apiKey
}

// ===== Manager 结构 =====

// FuturesManager Binance 期货统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

// NewFuturesManager 创建 Binance 期货管理器（凭证在此初始化一次）
func NewFuturesManager(apiKey, apiSecret, baseURL string) *FuturesManager {
	// 交易所级别的凭证只需初始化一次，然后所有业务客户端共享
	credentials := NewCredentials(apiKey, apiSecret)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return &FuturesManager{
		Order:    newFuturesOrderClient(credentials, httpClient, baseURL),
		Account:  newFuturesAccountClient(credentials, httpClient, baseURL),
		Position: newFuturesPositionClient(credentials, httpClient, baseURL),
	}
}

// SpotManager Binance 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Binance 现货管理器（凭证在此初始化一次）
func NewSpotManager(apiKey, apiSecret, baseURL string) *SpotManager {
	// 交易所级别的凭证只需初始化一次，然后所有业务客户端共享
	credentials := NewCredentials(apiKey, apiSecret)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return &SpotManager{
		Order:    newSpotOrderClient(credentials, httpClient, baseURL),
		Account:  newSpotAccountClient(credentials, httpClient, baseURL),
		Position: newSpotPositionClient(credentials, httpClient, baseURL),
	}
}

// ===== 内部工厂函数 =====

// newFuturesOrderClient 创建期货订单客户端
func newFuturesOrderClient(credentials *Credentials, httpClient *http.Client, baseURL string) *FuturesOrderClient {
	return &FuturesOrderClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}

// newFuturesAccountClient 创建期货账户客户端
func newFuturesAccountClient(credentials *Credentials, httpClient *http.Client, baseURL string) *FuturesAccountClient {
	return &FuturesAccountClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}

// newFuturesPositionClient 创建期货持仓客户端
func newFuturesPositionClient(credentials *Credentials, httpClient *http.Client, baseURL string) *FuturesPositionClient {
	return &FuturesPositionClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}

// newSpotOrderClient 创建现货订单客户端
func newSpotOrderClient(credentials *Credentials, httpClient *http.Client, baseURL string) *SpotOrderClient {
	return &SpotOrderClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}

// newSpotAccountClient 创建现货账户客户端
func newSpotAccountClient(credentials *Credentials, httpClient *http.Client, baseURL string) *SpotAccountClient {
	return &SpotAccountClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}

// newSpotPositionClient 创建现货持仓客户端
func newSpotPositionClient(credentials *Credentials, httpClient *http.Client, baseURL string) *SpotPositionClient {
	return &SpotPositionClient{
		credentials: credentials,
		httpClient:  httpClient,
		baseURL:     baseURL,
	}
}
