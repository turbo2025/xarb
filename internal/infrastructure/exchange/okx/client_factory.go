package okx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"
)

// ===== Credentials 凭证 =====

// Credentials 包含 OKX API 凭证和签名方法
type Credentials struct {
	apiKey     string
	apiSecret  string
	passphrase string
}

// NewCredentials 创建凭证对象
func NewCredentials(apiKey, apiSecret, passphrase string) *Credentials {
	return &Credentials{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
	}
}

// Sign 生成 OKX HMAC-SHA256 签名
// OKX 签名: BASE64(HMAC-SHA256(timestamp + method + requestPath + body, secretKey))
func (c *Credentials) Sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// APIKey 返回 API Key
func (c *Credentials) APIKey() string {
	return c.apiKey
}

// Passphrase 返回 Passphrase
func (c *Credentials) Passphrase() string {
	return c.passphrase
}

type managerDeps struct {
	credentials *Credentials
	httpClient  *http.Client
}

func newManagerDeps(apiKey, apiSecret, passphrase string) *managerDeps {
	return &managerDeps{
		credentials: NewCredentials(apiKey, apiSecret, passphrase),
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

// APIClient 封装访问 OKX REST API 所需的共享依赖
type APIClient struct {
	credentials *Credentials
	httpClient  *http.Client
	baseURL     string
}

// ===== Manager 结构 =====

// PerpetualManager OKX perpetual (perpetual contract) unified manager
type PerpetualManager struct {
	Account *PerpetualAccountClient
}

func newPerpetualManager(deps *managerDeps, perpetualURL string) *PerpetualManager {
	apiClient := deps.newAPIClient(perpetualURL)
	return &PerpetualManager{
		Account: NewPerpetualAccountClient(apiClient),
	}
}

// SpotManager OKX 现货统一管理器
type SpotManager struct {
	Account *SpotAccountClient
}

func newSpotManager(deps *managerDeps, spotURL string) *SpotManager {
	apiClient := deps.newAPIClient(spotURL)
	return &SpotManager{
		Account: NewSpotAccountClient(apiClient),
	}
}

// NewManagers 通过一组凭证和 URL 同时创建现货与perpetual manager
func NewManagers(apiKey, apiSecret, passphrase, spotURL, perpetualURL string) (*SpotManager, *PerpetualManager) {
	deps := newManagerDeps(apiKey, apiSecret, passphrase)
	spotMgr := newSpotManager(deps, spotURL)
	perpetualMgr := newPerpetualManager(deps, perpetualURL)
	return spotMgr, perpetualMgr
}
