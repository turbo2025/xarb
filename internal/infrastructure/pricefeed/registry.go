package pricefeed

import (
	"xarb/internal/application/port"

	"github.com/rs/zerolog/log"
)

// FactoryWithQuote 定义带quote参数的price feed factory函数类型
// wsURL: WebSocket连接URL
// quote: 计价货币 (e.g., "USDT")
type FactoryWithQuote func(wsURL string, quote string) port.PriceFeed

// registry maps exchange names to their respective price feed factories
var registry = make(map[string]FactoryWithQuote)

// Register 注册一个price feed factory for an exchange (使用新的FactoryWithQuote类型)
// 这是由各个交易所包的init()函数调用来自注册的
func Register(exchangeName string, factory FactoryWithQuote) {
	if factory == nil {
		log.Warn().Str("exchange", exchangeName).Msg("invalid price feed factory")
		return
	}
	if _, exists := registry[exchangeName]; exists {
		log.Warn().Str("exchange", exchangeName).Msg("price feed factory already registered, overwriting")
	}
	registry[exchangeName] = factory
	log.Debug().Str("exchange", exchangeName).Msg("price feed factory registered")
}

// Get 获取已注册的price feed factory for给定的exchange名称
func Get(exchangeName string) (FactoryWithQuote, bool) {
	factory, ok := registry[exchangeName]
	return factory, ok
}
