package bitget

import (
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Bitget perpetual WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Bitget
func init() {
	pricefeed.Register("bitget", func(wsURL string) port.PriceFeed {
		return NewPerpetualTickerFeed(wsURL)
	})
}
