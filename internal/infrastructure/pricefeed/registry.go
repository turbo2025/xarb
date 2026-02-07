package pricefeed

import (
	"xarb/internal/application/port"

	"github.com/rs/zerolog/log"
)

// Factory defines a price feed factory function type for creating exchange-specific price feeds
type Factory func(wsURL string) port.PriceFeed

// registry maps exchange names to their respective price feed factories
var registry = make(map[string]Factory)

// Register registers a price feed factory for an exchange
// This is called by each exchange package's init() function to self-register
func Register(exchangeName string, factory Factory) {
	if factory == nil {
		log.Warn().Str("exchange", exchangeName).Msg("invalid price feed factory")
		return
	}
	if _, exists := registry[exchangeName]; exists {
		log.Warn().Str("exchange", exchangeName).Msg("price feed factory already registered, overwriting")
	}
	registry[exchangeName] = factory
	log.Debug().Str("exchange", exchangeName).Msg("price feed factory registered")
}

// Get retrieves a registered price feed factory for the given exchange name
func Get(exchangeName string) (Factory, bool) {
	factory, ok := registry[exchangeName]
	return factory, ok
}
