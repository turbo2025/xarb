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

	if cfg.Exchange.Binance.Enabled {
		feeds = append(feeds, binance.NewFuturesMiniTickerFeed(cfg.Exchange.Binance.WsURL))
		log.Info().Msg("✓ Binance feed initialized")
	} else {
		log.Warn().Msg("⚠️ Binance disabled by config")
	}

	if cfg.Exchange.Bybit.Enabled {
		feeds = append(feeds, bybit.NewLinearTickerFeed(cfg.Exchange.Bybit.WsURL))
		log.Info().Msg("✓ Bybit feed initialized")
	} else {
		log.Warn().Msg("⚠️ Bybit disabled by config")
	}

	if cfg.Exchange.OKX.Enabled {
		feeds = append(feeds, okx.NewPublicLinearTickerFeed(cfg.Exchange.OKX.WsURL))
		log.Info().Msg("✓ OKX feed initialized")
	} else {
		log.Warn().Msg("⚠️ OKX disabled by config")
	}

	if cfg.Exchange.Bitget.Enabled {
		feeds = append(feeds, bitget.NewPublicMarketTickerFeed(cfg.Exchange.Bitget.WsURL))
		log.Info().Msg("✓ Bitget feed initialized")
	} else {
		log.Warn().Msg("⚠️ Bitget disabled by config")
	}

	return feeds
}
