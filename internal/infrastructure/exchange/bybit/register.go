package bybit

import (
	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers Bybit WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 Bybit
func init() {
	pricefeed.Register(application.ExchangeBybit, func(wsURL string) port.PriceFeed {
		return NewTickerFeed(wsURL)
	})
}
