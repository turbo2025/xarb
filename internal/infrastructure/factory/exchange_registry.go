package factory

import (
	"fmt"

	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
)

// BizSet 交易业务的客户端集合 (Order, Position, Account)
type BizSet struct {
	Order    interface{}
	Position interface{}
	Account  interface{}
}

// TradeTypeBizSets 交易所的交易类型业务集合
type TradeTypeBizSets struct {
	Spot    *BizSet
	Futures *BizSet
}

// ExchangeClientRegistry 交易所业务集合注册表
type ExchangeClientRegistry struct {
	clients map[string]*TradeTypeBizSets
}

// NewExchangeClientRegistry 创建交易所业务集合注册表
func NewExchangeClientRegistry() *ExchangeClientRegistry {
	return &ExchangeClientRegistry{
		clients: make(map[string]*TradeTypeBizSets),
	}
}

// newBizSetFromManager 从 Manager 创建 BizSet (内部辅助函数)
func newBizSetFromManager(order, position, account interface{}) *BizSet {
	return &BizSet{
		Order:    order,
		Position: position,
		Account:  account,
	}
}

// RegisterBinance 注册 Binance 业务集合
func (r *ExchangeClientRegistry) RegisterBinance(apiKey, apiSecret, futuresURL, spotURL string) {
	futuresMgr := binance.NewFuturesManager(apiKey, apiSecret, futuresURL)
	spotMgr := binance.NewSpotManager(apiKey, apiSecret, spotURL)

	r.clients[ExchangeBinance] = &TradeTypeBizSets{
		Futures: newBizSetFromManager(futuresMgr.Order, futuresMgr.Position, futuresMgr.Account),
		Spot:    newBizSetFromManager(spotMgr.Order, spotMgr.Position, spotMgr.Account),
	}
}

// RegisterBybit 注册 Bybit 业务集合
func (r *ExchangeClientRegistry) RegisterBybit(apiKey, apiSecret, futuresURL, spotURL string) {
	linearMgr := bybit.NewLinearManager(apiKey, apiSecret, futuresURL)
	spotMgr := bybit.NewSpotManager(apiKey, apiSecret, spotURL)

	r.clients[ExchangeBybit] = &TradeTypeBizSets{
		Futures: newBizSetFromManager(linearMgr.Order, linearMgr.Position, linearMgr.Account),
		Spot:    newBizSetFromManager(spotMgr.Order, spotMgr.Position, spotMgr.Account),
	}
}

// GetExchange 获取交易所业务集合
func (r *ExchangeClientRegistry) GetExchange(exchangeName string) (*TradeTypeBizSets, error) {
	if clients, ok := r.clients[exchangeName]; ok {
		return clients, nil
	}
	return nil, fmt.Errorf("exchange %s not registered", exchangeName)
}

// GetBizSet 获取指定交易所和交易类型的业务集合
func (r *ExchangeClientRegistry) GetBizSet(exchangeName, tradeType string) (*BizSet, error) {
	exchange, err := r.GetExchange(exchangeName)
	if err != nil {
		return nil, err
	}

	switch tradeType {
	case TradeTypeFutures:
		if exchange.Futures == nil {
			return nil, fmt.Errorf("%s does not have futures", exchangeName)
		}
		return exchange.Futures, nil
	case TradeTypeSpot:
		if exchange.Spot == nil {
			return nil, fmt.Errorf("%s does not have spot", exchangeName)
		}
		return exchange.Spot, nil
	default:
		return nil, fmt.Errorf("unknown trade type: %s", tradeType)
	}
}

// ListExchanges 列出所有已注册的交易所
func (r *ExchangeClientRegistry) ListExchanges() []string {
	var exchanges []string
	for name := range r.clients {
		exchanges = append(exchanges, name)
	}
	return exchanges
}

// HasExchange 检查交易所是否已注册
func (r *ExchangeClientRegistry) HasExchange(exchangeName string) bool {
	_, ok := r.clients[exchangeName]
	return ok
}
