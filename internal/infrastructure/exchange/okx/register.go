package okx

import (
	"fmt"
	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/pricefeed"
)

// init() automatically registers OKX perpetual WebSocket price feed factory
// 这样避免了在 factory.go 中硬编码 OKX
func init() {
	pricefeed.Register(application.ExchangeOKX, func(wsURL string, quote string) port.PriceFeed {

		return NewPerpetualTickerFeedWithQuote(wsURL, fmt.Sprintf("-%s-SWAP", quote))
	})
}
