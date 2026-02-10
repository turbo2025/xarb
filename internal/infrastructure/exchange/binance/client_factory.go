package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
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

// ===== Manager 结构 =====

// FuturesManager Binance 期货统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

// NewFuturesManager 创建 Binance 期货管理器（凭证在此初始化一次）
func NewFuturesManager(apiKey, apiSecret, baseURL string, httpClient *http.Client) *FuturesManager {
	credentials := NewCredentials(apiKey, apiSecret)
	return &FuturesManager{
		Order: &FuturesOrderClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
		Account: &FuturesAccountClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
		Position: &FuturesPositionClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
	}
}

// SpotManager Binance 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Binance 现货管理器（凭证在此初始化一次）
func NewSpotManager(apiKey, apiSecret, baseURL string, httpClient *http.Client) *SpotManager {
	credentials := NewCredentials(apiKey, apiSecret)
	return &SpotManager{
		Order: &SpotOrderClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
		Account: &SpotAccountClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
		Position: &SpotPositionClient{
			credentials: credentials,
			httpClient:  httpClient,
			baseURL:     baseURL,
		},
	}
}
