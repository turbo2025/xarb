package factory

import (
	"fmt"

	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/exchange/okx"
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

// BinancePerpetualClients Binance 永续合约客户端集合
type BinancePerpetualClients struct {
	Order    *binance.PerpetualOrderClient
	Position *binance.PerpetualPositionClient
	Account  *binance.PerpetualAccountClient
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

// BybitPerpetualClients Bybit 永续合约客户端集合
type BybitPerpetualClients struct {
	Order    *bybit.PerpetualOrderClient
	Position *bybit.PerpetualPositionClient
	Account  *bybit.PerpetualAccountClient
}

// ============================================
// OKX 客户端集合
// ============================================

// OKXSpotClients OKX 现货客户端集合
type OKXSpotClients struct {
	Account *okx.SpotAccountClient
}

// OKXPerpetualClients OKX 永续合约客户端集合
type OKXPerpetualClients struct {
	Account *okx.PerpetualAccountClient
}

// BitgetPerpetualClients Bitget 永续合约客户端集合（目前仅有 WebSocket 支持）
type BitgetPerpetualClients struct {
	// Bitget 目前仅实现 WebSocket 价格源，HTTP 客户端待实现
}

// ============================================
// 注册表
// ============================================

// ExchangeClientRegistry 交易所业务集合注册表
type ExchangeClientRegistry struct {
	binanceSpot      *BinanceSpotClients
	binancePerpetual *BinancePerpetualClients
	bybitSpot        *BybitSpotClients
	bybitPerpetual   *BybitPerpetualClients
	okxSpot          *OKXSpotClients
	okxPerpetual     *OKXPerpetualClients
	bitgetPerpetual  *BitgetPerpetualClients
}

// NewExchangeClientRegistry 创建交易所业务集合注册表
func NewExchangeClientRegistry() *ExchangeClientRegistry {
	return &ExchangeClientRegistry{}
}

// Register 通用注册方法，根据交易所名称动态注册
func (r *ExchangeClientRegistry) Register(exchangeName string, cfg *config.ExchangeConfig) error {
	// 验证参数
	if cfg.APIKey == "" || cfg.SecretKey == "" {
		return fmt.Errorf("%s: apiKey and apiSecret cannot be empty", exchangeName)
	}
	if cfg.PerpetualHttpURL == "" || cfg.SpotHttpURL == "" {
		return fmt.Errorf("%s: PerpetualHttpURL and SpotHttpURL cannot be empty", exchangeName)
	}
	switch exchangeName {
	case ExchangeBinance:
		return r.RegisterBinance(cfg)
	case ExchangeBybit:
		return r.RegisterBybit(cfg)
	case ExchangeOKX:
		return r.RegisterOKX(cfg)
	case ExchangeBitget:
		return r.RegisterBitget(cfg)
	default:
		return fmt.Errorf("unknown exchange: %s", exchangeName)
	}
}

// RegisterBinance 注册 Binance 业务集合
func (r *ExchangeClientRegistry) RegisterBinance(cfg *config.ExchangeConfig) error {

	// 检查是否已注册
	if r.binanceSpot != nil || r.binancePerpetual != nil {
		return fmt.Errorf("binance: already registered")
	}

	// 创建两个 Manager（内部会共享 HTTP 客户端与凭证）
	spotMgr, perpetualMgr := binance.NewManagers(cfg.APIKey, cfg.SecretKey, cfg.SpotHttpURL, cfg.PerpetualHttpURL)

	// 组装成强类型的客户端集合
	r.binanceSpot = &BinanceSpotClients{
		Order:    spotMgr.Order,
		Position: spotMgr.Position,
		Account:  spotMgr.Account,
	}

	r.binancePerpetual = &BinancePerpetualClients{
		Order:    perpetualMgr.Order,
		Position: perpetualMgr.Position,
		Account:  perpetualMgr.Account,
	}

	return nil
}

// RegisterBybit 注册 Bybit 业务集合
func (r *ExchangeClientRegistry) RegisterBybit(cfg *config.ExchangeConfig) error {
	// 检查是否已注册
	if r.bybitSpot != nil || r.bybitPerpetual != nil {
		return fmt.Errorf("bybit: already registered")
	}

	// 创建两个 Manager（内部会共享 HTTP 客户端与凭证）
	spotMgr, perpetualMgr := bybit.NewManagers(cfg.APIKey, cfg.SecretKey, cfg.SpotHttpURL, cfg.PerpetualHttpURL)

	// 组装成强类型的客户端集合
	r.bybitSpot = &BybitSpotClients{
		Order:    spotMgr.Order,
		Position: spotMgr.Position,
		Account:  spotMgr.Account,
	}

	r.bybitPerpetual = &BybitPerpetualClients{
		Order:    perpetualMgr.Order,
		Position: perpetualMgr.Position,
		Account:  perpetualMgr.Account,
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

// BinancePerpetual 获取 Binance 永续合约客户端集合
func (r *ExchangeClientRegistry) BinancePerpetual() *BinancePerpetualClients {
	return r.binancePerpetual
}

// BybitSpot 获取 Bybit 现货客户端集合
func (r *ExchangeClientRegistry) BybitSpot() *BybitSpotClients {
	return r.bybitSpot
}

// BybitPerpetual 获取 Bybit 永续合约客户端集合
func (r *ExchangeClientRegistry) BybitPerpetual() *BybitPerpetualClients {
	return r.bybitPerpetual
}

// RegisterOKX 注册 OKX 业务集合
func (r *ExchangeClientRegistry) RegisterOKX(cfg *config.ExchangeConfig) error {
	// 检查是否已注册
	if r.okxSpot != nil || r.okxPerpetual != nil {
		return fmt.Errorf("okx: already registered")
	}

	// OKX 需要 Passphrase，检查它
	if cfg.Passphrase == "" {
		return fmt.Errorf("okx: passphrase cannot be empty")
	}

	// 创建两个 Manager（内部会共享 HTTP 客户端与凭证）
	spotMgr, perpetualMgr := okx.NewManagers(cfg.APIKey, cfg.SecretKey, cfg.Passphrase, cfg.SpotHttpURL, cfg.PerpetualHttpURL)

	// 组装成强类型的客户端集合
	r.okxSpot = &OKXSpotClients{
		Account: spotMgr.Account,
	}

	r.okxPerpetual = &OKXPerpetualClients{
		Account: perpetualMgr.Account,
	}

	return nil
}

// OKXSpot 获取 OKX 现货客户端集合
func (r *ExchangeClientRegistry) OKXSpot() *OKXSpotClients {
	return r.okxSpot
}

// OKXPerpetual 获取 OKX 永续合约客户端集合
func (r *ExchangeClientRegistry) OKXPerpetual() *OKXPerpetualClients {
	return r.okxPerpetual
}

// RegisterBitget 注册 Bitget 业务集合（目前仅支持 WebSocket）
func (r *ExchangeClientRegistry) RegisterBitget(cfg *config.ExchangeConfig) error {
	// 检查是否已注册
	if r.bitgetPerpetual != nil {
		return fmt.Errorf("bitget: already registered")
	}

	// Bitget 目前仅实现 WebSocket 价格源，HTTP 客户端待实现
	// 此处预留位置，确保 bitget 包被导入以执行其 init() 函数注册 pricefeed 工厂
	r.bitgetPerpetual = &BitgetPerpetualClients{}

	return nil
}

// BitgetPerpetual 获取 Bitget 永续合约客户端集合
func (r *ExchangeClientRegistry) BitgetPerpetual() *BitgetPerpetualClients {
	return r.bitgetPerpetual
}
