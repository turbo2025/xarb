package factory

import (
	"fmt"

	"xarb/internal/infrastructure/config"
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
	Order    *bybit.FuturesOrderClient
	Position *bybit.FuturesPositionClient
	Account  *bybit.FuturesAccountClient
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
func (r *ExchangeClientRegistry) Register(exchangeName string, cfg config.ExchangeConfig) error {
	// 验证参数
	if cfg.APIKey == "" || cfg.SecretKey == "" {
		return fmt.Errorf("%s: apiKey and apiSecret cannot be empty", exchangeName)
	}
	if cfg.FuturesURL == "" || cfg.SpotURL == "" {
		return fmt.Errorf("%s: futuresURL and spotURL cannot be empty", exchangeName)
	}
	switch exchangeName {
	case ExchangeBinance:
		return r.RegisterBinance(cfg)
	case ExchangeBybit:
		return r.RegisterBybit(cfg)
	default:
		return fmt.Errorf("unknown exchange: %s", exchangeName)
	}
}

// RegisterBinance 注册 Binance 业务集合
func (r *ExchangeClientRegistry) RegisterBinance(cfg config.ExchangeConfig) error {

	// 检查是否已注册
	if r.binanceSpot != nil || r.binanceFutures != nil {
		return fmt.Errorf("binance: already registered")
	}

	// 创建 Manager 配置（自动初始化 HTTP 连接、凭证和 URL）
	config := binance.NewManagerConfig(cfg)

	// 创建两个 Manager
	spotMgr := binance.NewSpotManager(config)
	futuresMgr := binance.NewFuturesManager(config)

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
func (r *ExchangeClientRegistry) RegisterBybit(cfg config.ExchangeConfig) error {
	// 检查是否已注册
	if r.bybitSpot != nil || r.bybitFutures != nil {
		return fmt.Errorf("bybit: already registered")
	}

	// 创建 Manager 配置（自动初始化 HTTP 连接、凭证和 URL）
	config := bybit.NewManagerConfig(cfg)

	// 创建两个 Manager
	spotMgr := bybit.NewSpotManager(config)
	futuresMgr := bybit.NewFuturesManager(config)

	// 组装成强类型的客户端集合
	r.bybitSpot = &BybitSpotClients{
		Order:    spotMgr.Order,
		Position: spotMgr.Position,
		Account:  spotMgr.Account,
	}

	r.bybitFutures = &BybitFuturesClients{
		Order:    futuresMgr.Order,
		Position: futuresMgr.Position,
		Account:  futuresMgr.Account,
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
