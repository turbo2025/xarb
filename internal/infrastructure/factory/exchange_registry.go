package factory

import (
	"fmt"
	"net/http"
	"time"

	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
)

// ============================================
// Binance 客户端集合
// ============================================

// BinanceSpotClients Binance 现货客户端集合
type BinanceSpotClients struct {
	Order    *binance.SpotOrderClient
	Position *binance.SpotPositionClient
	Account  *binance.SpotAccountClient
}

// BinanceFuturesClients Binance 期货客户端集合
type BinanceFuturesClients struct {
	Order    *binance.FuturesOrderClient
	Position *binance.FuturesPositionClient
	Account  *binance.FuturesAccountClient
}

// ============================================
// Bybit 客户端集合
// ============================================

// BybitSpotClients Bybit 现货客户端集合
type BybitSpotClients struct {
	Order    *bybit.SpotOrderClient
	Position *bybit.SpotPositionClient
	Account  *bybit.SpotAccountClient
}

// BybitFuturesClients Bybit 期货客户端集合
type BybitFuturesClients struct {
	Order    *bybit.LinearOrderClient
	Position *bybit.LinearPositionClient
	Account  *bybit.LinearAccountClient
}

// ============================================
// 注册表
// ============================================

// ExchangeClientRegistry 交易所业务集合注册表
type ExchangeClientRegistry struct {
	binanceSpot    *BinanceSpotClients
	binanceFutures *BinanceFuturesClients
	bybitSpot      *BybitSpotClients
	bybitFutures   *BybitFuturesClients
}

// NewExchangeClientRegistry 创建交易所业务集合注册表
func NewExchangeClientRegistry() *ExchangeClientRegistry {
	return &ExchangeClientRegistry{}
}

// Register 通用注册方法，根据交易所名称动态注册
func (r *ExchangeClientRegistry) Register(exchangeName, apiKey, apiSecret, futuresURL, spotURL string) error {
	// 验证参数
	if apiKey == "" || apiSecret == "" {
		return fmt.Errorf("binance: apiKey and apiSecret cannot be empty")
	}
	if futuresURL == "" || spotURL == "" {
		return fmt.Errorf("binance: futuresURL and spotURL cannot be empty")
	}
	switch exchangeName {
	case ExchangeBinance:
		return r.RegisterBinance(apiKey, apiSecret, futuresURL, spotURL)
	case ExchangeBybit:
		return r.RegisterBybit(apiKey, apiSecret, futuresURL, spotURL)
	default:
		return fmt.Errorf("unknown exchange: %s", exchangeName)
	}
}

// RegisterBinance 注册 Binance 业务集合
func (r *ExchangeClientRegistry) RegisterBinance(apiKey, apiSecret, futuresURL, spotURL string) error {

	// 检查是否已注册
	if r.binanceSpot != nil || r.binanceFutures != nil {
		return fmt.Errorf("binance: already registered")
	}
	// 为该交易所创建共享的 HTTP 连接池
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建两个 Manager（共享 httpClient 和凭证）
	spotMgr := binance.NewSpotManager(apiKey, apiSecret, spotURL, httpClient)
	futuresMgr := binance.NewFuturesManager(apiKey, apiSecret, futuresURL, httpClient)

	// 组装成强类型的客户端集合
	r.binanceSpot = &BinanceSpotClients{
		Order:    spotMgr.Order,
		Position: spotMgr.Position,
		Account:  spotMgr.Account,
	}

	r.binanceFutures = &BinanceFuturesClients{
		Order:    futuresMgr.Order,
		Position: futuresMgr.Position,
		Account:  futuresMgr.Account,
	}

	return nil
}

// RegisterBybit 注册 Bybit 业务集合
func (r *ExchangeClientRegistry) RegisterBybit(apiKey, apiSecret, futuresURL, spotURL string) error {
	// 检查是否已注册
	if r.bybitSpot != nil || r.bybitFutures != nil {
		return fmt.Errorf("bybit: already registered")
	}

	// 为该交易所创建共享的 HTTP 连接池
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	linearMgr := bybit.NewLinearManager(apiKey, apiSecret, futuresURL, httpClient)
	spotMgr := bybit.NewSpotManager(apiKey, apiSecret, spotURL, httpClient)

	r.bybitSpot = &BybitSpotClients{
		Order:    spotMgr.Order,
		Position: spotMgr.Position,
		Account:  spotMgr.Account,
	}

	r.bybitFutures = &BybitFuturesClients{
		Order:    linearMgr.Order,
		Position: linearMgr.Position,
		Account:  linearMgr.Account,
	}

	return nil
}

// ============================================
// 快速访问方法
// ============================================

// BinanceSpot 获取 Binance 现货客户端集合
func (r *ExchangeClientRegistry) BinanceSpot() *BinanceSpotClients {
	return r.binanceSpot
}

// BinanceFutures 获取 Binance 期货客户端集合
func (r *ExchangeClientRegistry) BinanceFutures() *BinanceFuturesClients {
	return r.binanceFutures
}

// BybitSpot 获取 Bybit 现货客户端集合
func (r *ExchangeClientRegistry) BybitSpot() *BybitSpotClients {
	return r.bybitSpot
}

// BybitFutures 获取 Bybit 期货客户端集合
func (r *ExchangeClientRegistry) BybitFutures() *BybitFuturesClients {
	return r.bybitFutures
}
