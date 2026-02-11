package bybit

import (
	"net/http"
	"time"
	"xarb/internal/infrastructure/config"
)

// ClientFields 共享客户端字段
type ClientFields struct {
	ApiKey     string
	ApiSecret  string
	HttpClient *http.Client
	BaseURL    string
}

// NewClientFields 创建共享的客户端字段
func NewClientFields(apiKey, apiSecret string, httpClient *http.Client) *ClientFields {
	return &ClientFields{
		ApiKey:     apiKey,
		ApiSecret:  apiSecret,
		HttpClient: httpClient,
	}
}

// ManagerConfig Bybit Manager 配置（集中管理 HTTP 连接、凭证和 URL）
type ManagerConfig struct {
	fields     *ClientFields
	SpotURL    string
	FuturesURL string
}

// NewManagerConfig 创建 Bybit Manager 配置（自动初始化 HTTP 连接）
func NewManagerConfig(cfg config.ExchangeConfig) *ManagerConfig {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	fields := NewClientFields(cfg.APIKey, cfg.SecretKey, httpClient)
	return &ManagerConfig{
		fields:     fields,
		SpotURL:    cfg.SpotURL,
		FuturesURL: cfg.FuturesURL,
	}
}

// ===== Manager 结构 =====

// FuturesManager Bybit 期货（perpetual）统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

// NewFuturesManager 创建 Bybit 期货管理器（从 Manager 配置中读取 URL）
func NewFuturesManager(cfg *ManagerConfig) *FuturesManager {
	return &FuturesManager{
		Order: &FuturesOrderClient{
			ClientFields: cfg.fields,
			baseURL:      cfg.FuturesURL,
		},
		Account: &FuturesAccountClient{
			ClientFields: cfg.fields,
		},
		Position: &FuturesPositionClient{
			ClientFields: cfg.fields,
		},
	}
}

// SpotManager Bybit 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

// NewSpotManager 创建 Bybit 现货管理器（从 Manager 配置中读取 URL）
func NewSpotManager(cfg *ManagerConfig) *SpotManager {
	return &SpotManager{
		Order: &SpotOrderClient{
			ClientFields: cfg.fields,
			baseURL:      cfg.SpotURL,
		},
		Account: &SpotAccountClient{
			ClientFields: cfg.fields,
		},
		Position: &SpotPositionClient{
			ClientFields: cfg.fields,
		},
	}
}

// ===== 别名和辅助函数 =====
