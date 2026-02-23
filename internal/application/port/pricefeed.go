package port

import "context"

type Tick struct {
	Exchange string  // 交易所 "BINANCE" "BYBIT"
	Symbol   string  // "BTCUSDT"
	PriceStr string  // raw string
	PriceNum float64 // parsed float64 (best-effort)
	Ts       int64   // unix ms
}

type PriceFeed interface {
	Name() string
	Subscribe(ctx context.Context, coins []string) (<-chan Tick, error)
	// Symbol2Coin 将交易所特定格式的交易对转换为币种
	// 例: BTC-USDT-SWAP -> BTC, BTCUSDT -> BTC
	Symbol2Coin(symbol string) string
	// Coin2Symbol 将币种转换为交易所特定格式的交易对
	// 例: BTC -> BTC-USDT-SWAP (OKX), BTC -> BTCUSDT (Bybit)
	Coin2Symbol(coin string) string
}
