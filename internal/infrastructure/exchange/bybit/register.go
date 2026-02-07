package bybit

import (
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Bybit perpetual WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Bybit
func init() {
	pricefeed.Register("bybit", func(wsURL string) port.PriceFeed {
		return NewPerpetualTickerFeed(wsURL)
	})
}
