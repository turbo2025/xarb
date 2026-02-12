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

type APIClient struct {
	credentials *Credentials
	httpClient  *http.Client
	baseURL     string
}

// managerDeps 在内部复用 HTTP 连接与凭证
type managerDeps struct {
	credentials *Credentials
	httpClient  *http.Client
}

func newManagerDeps(apiKey, apiSecret string) *managerDeps {
	return &managerDeps{
		credentials: NewCredentials(apiKey, apiSecret),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *managerDeps) newAPIClient(baseURL string) *APIClient {
	return &APIClient{
		credentials: d.credentials,
		httpClient:  d.httpClient,
		baseURL:     baseURL,
	}
}

// FuturesManager Binance 期货统一管理器
type FuturesManager struct {
	Order    *FuturesOrderClient
	Account  *FuturesAccountClient
	Position *FuturesPositionClient
}

func newFuturesManager(deps *managerDeps, futuresURL string) *FuturesManager {
	apiClient := deps.newAPIClient(futuresURL)
	return &FuturesManager{
		Order:    NewFuturesOrderClient(apiClient),
		Account:  NewFuturesAccountClient(apiClient),
		Position: NewFuturesPositionClient(apiClient),
	}
}

// SpotManager Binance 现货统一管理器
type SpotManager struct {
	Order    *SpotOrderClient
	Account  *SpotAccountClient
	Position *SpotPositionClient
}

func newSpotManager(deps *managerDeps, spotURL string) *SpotManager {
	apiClient := deps.newAPIClient(spotURL)
	return &SpotManager{
		Order:    NewSpotOrderClient(apiClient),
		Account:  NewSpotAccountClient(apiClient),
		Position: NewSpotPositionClient(apiClient),
	}
}

// NewManagers 通过一组凭证和 URL 同时创建现货与期货管理器
func NewManagers(apiKey, apiSecret, spotURL, futuresURL string) (*SpotManager, *FuturesManager) {
	deps := newManagerDeps(apiKey, apiSecret)
	spotMgr := newSpotManager(deps, spotURL)
	futuresMgr := newFuturesManager(deps, futuresURL)
	return spotMgr, futuresMgr
}
