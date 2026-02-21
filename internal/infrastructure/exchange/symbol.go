package exchange

import (
	"strings"
)

// SymbolConverter 符号转换接口
// 各交易所可以实现此接口来提供符号转换功能
type SymbolConverter interface {
	// Symbol2Coin 将交易对转换为币种
	// 例: BTCUSDT -> BTC, BTC-USDT-SWAP -> BTC
	Symbol2Coin(symbol string) string

	// Coin2Symbol 将币种转换为交易对
	// 例: BTC -> BTCUSDT
	Coin2Symbol(coin string) string

	// SymbolSuffix 返回符号后缀
	// 例: USDT, USDC, -USDT-SWAP 等
	SymbolSuffix() string
}

// CommonSymbolConverter 通用符号转换器
type CommonSymbolConverter struct {
	suffix string
}

// NewCommonSymbolConverter 创建通用符号转换器
func NewCommonSymbolConverter(suffix string) *CommonSymbolConverter {
	return &CommonSymbolConverter{suffix: strings.ToUpper(strings.TrimSpace(suffix))}
}

// SymbolSuffix 返回符号后缀
func (c *CommonSymbolConverter) SymbolSuffix() string {
	return c.suffix
}

// Symbol2Coin 将交易对转换为币种
// 例: BTCUSDT -> BTC, BTC-USDT-SWAP -> BTC, BTCUSDC -> BTC
func (c *CommonSymbolConverter) Symbol2Coin(symbol string) string {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return ""
	}
	sym = strings.ReplaceAll(sym, c.suffix, "")

	// 移除 1\d+ 模式 (比如 1000, 1100 等)
	// 这用于去除杠杆倍数标记
	// re := regexp.MustCompile(`1\d+`)
	// sym = re.ReplaceAllString(sym, "")

	return sym
}

// Coin2Symbol 将币种转换为交易对
// 例: BTC -> BTCUSDT, BTCUSDT -> BTCUSDT
func (c *CommonSymbolConverter) Coin2Symbol(coin string) string {
	coin = strings.ToUpper(strings.TrimSpace(coin))
	if coin == "" {
		return ""
	}

	// 如果已经包含后缀，直接返回
	if strings.HasSuffix(coin, c.suffix) {
		return coin
	}
	// 否则添加后缀
	return coin + c.suffix
}
