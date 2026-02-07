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

// PerpetualManager Binance perpetual unified manager
type PerpetualManager struct {
	Order    *PerpetualOrderClient
	Account  *PerpetualAccountClient
	Position *PerpetualPositionClient
}

func newPerpetualManager(deps *managerDeps, perpetualURL string) *PerpetualManager {
	apiClient := deps.newAPIClient(perpetualURL)
	return &PerpetualManager{
		Order:    NewPerpetualOrderClient(apiClient),
		Account:  NewPerpetualAccountClient(apiClient),
		Position: NewPerpetualPositionClient(apiClient),
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

// NewManagers 通过一组凭证和 URL 同时创建现货与perpetual manager
func NewManagers(apiKey, apiSecret, spotURL, perpetualURL string) (*SpotManager, *PerpetualManager) {
	deps := newManagerDeps(apiKey, apiSecret)
	spotMgr := newSpotManager(deps, spotURL)
	perpetualMgr := newPerpetualManager(deps, perpetualURL)
	return spotMgr, perpetualMgr
}
