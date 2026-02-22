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
}
