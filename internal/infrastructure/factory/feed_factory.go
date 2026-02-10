package factory

import (
	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bitget"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/exchange/okx"

	"github.com/rs/zerolog/log"
)

// NewPriceFeeds 初始化交易所数据源
func NewPriceFeeds(cfg *config.Config) []monitor.PriceFeed {
	var feeds []monitor.PriceFeed

	// 遍历所有启用的交易所并初始化数据源
	for exchangeName, exchCfg := range cfg.Exchanges {
		if !exchCfg.Enabled {
			log.Warn().Msgf("⚠️ %s disabled by config", exchangeName)
			continue
		}

		var feed monitor.PriceFeed
		switch exchangeName {
		case "binance":
			feed = binance.NewFuturesMiniTickerFeed(exchCfg.WsURL)
		case "bybit":
			feed = bybit.NewLinearTickerFeed(exchCfg.WsURL)
		case "okx":
			feed = okx.NewPublicLinearTickerFeed(exchCfg.WsURL)
		case "bitget":
			feed = bitget.NewPublicMarketTickerFeed(exchCfg.WsURL)
		default:
			log.Warn().Msgf("⚠️ Unknown exchange: %s", exchangeName)
			continue
		}

		feeds = append(feeds, feed)
		log.Info().Msgf("✓ %s feed initialized", exchangeName)
	}

	return feeds
}
