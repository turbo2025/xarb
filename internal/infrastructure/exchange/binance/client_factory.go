package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
	"xarb/internal/infrastructure/config"
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

// ManagerConfig Binance Manager 配置（集中管理 HTTP 连接、凭证和 URL）
type ManagerConfig struct {
	credentials *Credentials
	httpClient  *http.Client
	SpotURL     string
	FuturesURL  string
}

// NewManagerConfig 创建 Binance Manager 配置（自动初始化 HTTP 连接）
func NewManagerConfig(cfg config.ExchangeConfig) *ManagerConfig {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	credentials := NewCredentials(cfg.APIKey, cfg.SecretKey)
	return &ManagerConfig{
		credentials: credentials,
		httpClient:  httpClient,
		SpotURL:     cfg.SpotURL,
		FuturesURL:  cfg.FuturesURL,
	}
}

// ===== Manager 结构 =====

// ===== Manager 结构 =====

// FuturesManager Binance 期货统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

// NewFuturesManager 创建 Binance 期货管理器（从 Manager 配置中读取参数）
func NewFuturesManager(cfg *ManagerConfig) *FuturesManager {
	return &FuturesManager{
		Order: &FuturesOrderClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.FuturesURL,
		},
		Account: &FuturesAccountClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.FuturesURL,
		},
		Position: &FuturesPositionClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.FuturesURL,
		},
	}
}

// SpotManager Binance 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Binance 现货管理器（从 Manager 配置中读取参数）
func NewSpotManager(cfg *ManagerConfig) *SpotManager {
	return &SpotManager{
		Order: &SpotOrderClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.SpotURL,
		},
		Account: &SpotAccountClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.SpotURL,
		},
		Position: &SpotPositionClient{
			credentials: cfg.credentials,
			httpClient:  cfg.httpClient,
			baseURL:     cfg.SpotURL,
		},
	}
}
