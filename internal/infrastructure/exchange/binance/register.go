package binance

import (
	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Binance WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Binance
func init() {
	pricefeed.Register(application.ExchangeBinance, func(wsURL string) port.PriceFeed {
		return NewTickerFeed(wsURL)
	})
}
