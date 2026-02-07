package okx

import (
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers OKX perpetual WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 OKX
func init() {
	pricefeed.Register("okx", func(wsURL string) port.PriceFeed {
		return NewPerpetualTickerFeed(wsURL)
	})
}
