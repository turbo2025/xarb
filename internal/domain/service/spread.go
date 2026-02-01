package service

func Delta(bybit, binance float64) float64 {
	return bybit - binance
}

func DeltaColor(delta, threshold float64) int {
	// -1 red, 0 yellow, +1 green (pure decision)
	if delta >= threshold {
		return +1
	}
	if delta <= -threshold {
		return -1
	}
	return 0
}
