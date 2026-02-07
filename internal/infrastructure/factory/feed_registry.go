package factory

import (
	"xarb/internal/application/usecase/monitor"

	"github.com/rs/zerolog/log"
)

// PriceFeedFactory 定义交易所价格源工厂函数类型
type PriceFeedFactory func(wsURL string) monitor.PriceFeed

// priceFeeds 交易所名称 -> 价格源工厂函数映射
// 各交易所包通过 RegisterPriceFeed() 自动注册，避免硬编码
var priceFeeds = make(map[string]PriceFeedFactory)

// RegisterPriceFeed 注册价格源工厂函数（由各交易所包调用）
// 这允许各交易所包在自己的 register.go 中自动注册，无需修改此文件
func RegisterPriceFeed(exchangeName string, factory PriceFeedFactory) {
	if factory == nil {
		log.Warn().Str("exchange", exchangeName).Msg("invalid price feed factory")
		return
	}
	if _, exists := priceFeeds[exchangeName]; exists {
		log.Warn().Str("exchange", exchangeName).Msg("price feed factory already registered, overwriting")
	}
	priceFeeds[exchangeName] = factory
	log.Debug().Str("exchange", exchangeName).Msg("price feed factory registered")
}

// GetPriceFeed 获取已注册的价格源工厂
func GetPriceFeed(exchangeName string) (PriceFeedFactory, bool) {
	factory, ok := priceFeeds[exchangeName]
	return factory, ok
}
