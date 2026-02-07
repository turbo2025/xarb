package binance

import (
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Binance perpetual WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Binance
func init() {
	pricefeed.Register("binance", func(wsURL string) port.PriceFeed {
		return NewPerpetualTickerFeed(wsURL)
	})
}
