package factory

import (
	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/pricefeed"

	"github.com/rs/zerolog/log"
)

// NewPriceFeeds 初始化交易所数据源
// 从配置中获取 enabled 的交易所列表，使用已注册的工厂函数动态初始化价格源
// 各交易所的 websocket 工厂在其 register.go 中自动注册，无需在此硬编码
// 注意：exchange 包会通过 api client factory 初始化时自动导入，
// 其 init() 函数会自动向 pricefeed.registry 注册 websocket 工厂
func NewPriceFeeds(cfg *config.Config) []monitor.PriceFeed {
	var feeds []monitor.PriceFeed

	// 获取所有 enabled 的交易所（避免硬编码和重复检查）
	enabledExchanges := cfg.GetEnabledExchanges()

	// 只遍历 enabled 的交易所，使用已注册的工厂函数动态初始化
	for _, exchangeName := range enabledExchanges {
		exchCfg := cfg.Exchanges[exchangeName]

		// 从注册表中获取工厂函数（避免硬编码 switch，工厂由各交易所包自己注册）
		factory, ok := pricefeed.Get(exchangeName)
		if !ok {
			log.Warn().Msgf("⚠️ Unknown exchange or price feed not registered: %s", exchangeName)
			continue
		}

		// 动态调用工厂函数创建价格源
		feed := factory(exchCfg.PerpetualWsURL)
		feeds = append(feeds, feed)
		log.Info().Msgf("✓ %s feed initialized", exchangeName)
	}

	return feeds
}
