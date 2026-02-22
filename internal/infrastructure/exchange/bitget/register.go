package bitget

import (
	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Bitget WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Bitget
func init() {
	pricefeed.Register(application.ExchangeBitget, func(wsURL string) port.PriceFeed {
		return NewTickerFeed(wsURL)
	})
}
