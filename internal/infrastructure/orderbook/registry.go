package orderbook

import "github.com/rs/zerolog/log"

// OrderBook 定义订单簿接口（为未来使用）
// 具体实现在各交易所包中
type OrderBook interface {
	// 具体方法将根据需要添加
}

// Factory 定义订单簿工厂函数类型
type Factory func(wsURL string) OrderBook

// registry 订单簿工厂注册表
var registry = make(map[string]Factory)

// Register 注册订单簿工厂（由各交易所包的 order_book_register.go 调用）
func Register(exchangeName string, factory Factory) {
	if factory == nil {
		log.Warn().Str("exchange", exchangeName).Msg("invalid order book factory")
		return
	}
	if _, exists := registry[exchangeName]; exists {
		log.Warn().Str("exchange", exchangeName).Msg("order book factory already registered, overwriting")
	}
	registry[exchangeName] = factory
	log.Debug().Str("exchange", exchangeName).Msg("order book factory registered")
}

// Get 获取已注册的订单簿工厂
func Get(exchangeName string) (Factory, bool) {
	factory, ok := registry[exchangeName]
	return factory, ok
}
