package service

import (
	"fmt"
	"strings"
	"sync"
)

// SymbolMapper 将交易所特定符号映射到规范符号
// 例如：Binance BTCUSDT -> 规范 BTC/USDT，OKX BTC-USDT-SWAP -> 规范 BTC/USDT
type SymbolMapper struct {
	mu               sync.RWMutex
	canonicalSymbols map[string]string   // exchange:symbol -> canonical (e.g., "binance:btcusdt" -> "BTC/USDT")
	reverseMap       map[string][]string // canonical -> [exchange:symbols] (for reverse lookup)
}

// NewSymbolMapper 创建新的符号映射器
func NewSymbolMapper() *SymbolMapper {
	return &SymbolMapper{
		canonicalSymbols: make(map[string]string),
		reverseMap:       make(map[string][]string),
	}
}

// Register 注册交易所符号与规范符号的映射
// exchange: 交易所名称 (e.g., "BINANCE", "OKX")
// symbol: 交易所的原始符号 (e.g., "BTCUSDT", "BTC-USDT-SWAP")
// canonical: 规范符号 (e.g., "BTC/USDT")
func (m *SymbolMapper) Register(exchange, symbol, canonical string) error {
	if exchange == "" || symbol == "" || canonical == "" {
		return fmt.Errorf("exchange, symbol, and canonical cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(exchange, symbol)
	m.canonicalSymbols[key] = canonical

	// 建立反向映射
	if m.reverseMap[canonical] == nil {
		m.reverseMap[canonical] = []string{}
	}
	// 避免重复
	found := false
	for _, k := range m.reverseMap[canonical] {
		if k == key {
			found = true
			break
		}
	}
	if !found {
		m.reverseMap[canonical] = append(m.reverseMap[canonical], key)
	}

	return nil
}

// ToCanonical 将交易所符号转换为规范符号
func (m *SymbolMapper) ToCanonical(exchange, symbol string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.makeKey(exchange, symbol)
	canonical, ok := m.canonicalSymbols[key]
	if !ok {
		return "", fmt.Errorf("no mapping found for %s:%s", exchange, symbol)
	}
	return canonical, nil
}

// GetSymbolsForCanonical 获取某个规范符号在所有交易所中的实际符号
func (m *SymbolMapper) GetSymbolsForCanonical(canonical string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	keys, ok := m.reverseMap[canonical]
	if !ok {
		return result
	}

	for _, key := range keys {
		exchange, symbol := m.parseKey(key)
		// 交易所名称使用原始大小写，符号也使用原始大小写
		result[strings.ToUpper(exchange)] = symbol
	}
	return result
}

// LoadFromConfig 从配置中加载符号映射
// config 格式: map[canonical]map[exchange]symbol
// 例如: {"BTC/USDT": {"BINANCE": "BTCUSDT", "OKX": "BTC-USDT-SWAP"}}
func (m *SymbolMapper) LoadFromConfig(config map[string]map[string]string) error {
	for canonical, exchanges := range config {
		for exchange, symbol := range exchanges {
			if err := m.Register(exchange, symbol, canonical); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportConfig 导出当前映射为配置格式
func (m *SymbolMapper) ExportConfig() map[string]map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]string)
	for canonical, keys := range m.reverseMap {
		if result[canonical] == nil {
			result[canonical] = make(map[string]string)
		}
		for _, key := range keys {
			exchange, symbol := m.parseKey(key)
			result[canonical][exchange] = symbol
		}
	}
	return result
}

// makeKey 创建查找键
func (m *SymbolMapper) makeKey(exchange, symbol string) string {
	return strings.ToLower(exchange) + ":" + strings.ToUpper(symbol)
}

// parseKey 解析查找键
func (m *SymbolMapper) parseKey(key string) (exchange, symbol string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// RegisterMultiQuote 为多种结算货币注册符号
// 例如：RegisterMultiQuote("BINANCE", "BTC", []string{"USDT", "USDC"})
// 会注册 BTC/USDT 和 BTC/USDC
func (m *SymbolMapper) RegisterMultiQuote(exchange, baseAsset string, quotes []string) error {
	for _, quote := range quotes {
		// 构建交易所符号（例如 BTCUSDT、BTCUSDC）
		exchangeSymbol := strings.ToUpper(baseAsset + quote)
		// 构建规范符号（例如 BTC/USDT、BTC/USDC）
		canonical := strings.ToUpper(baseAsset) + "/" + strings.ToUpper(quote)

		if err := m.Register(exchange, exchangeSymbol, canonical); err != nil {
			return err
		}
	}
	return nil
}

// RegisterMultiExchange 为多个交易所注册相同符号
// 例如：RegisterMultiExchange([]string{"BINANCE", "BYBIT"}, "BTCUSDT", "BTC/USDT")
func (m *SymbolMapper) RegisterMultiExchange(exchanges []string, symbol, canonical string) error {
	for _, exchange := range exchanges {
		if err := m.Register(exchange, symbol, canonical); err != nil {
			return err
		}
	}
	return nil
}

// LoadDefaultConfig 加载默认的多交易所、多结算货币配置
func (m *SymbolMapper) LoadDefaultConfig() error {
	defaultConfig := map[string]map[string]string{
		"BTC/USDT": {
			"BINANCE": "BTCUSDT",
			"BYBIT":   "BTCUSDT",
			"OKX":     "BTC-USDT-SWAP",
			"BITGET":  "BTCUSDT",
		},
		"BTC/USDC": {
			"BINANCE": "BTCUSDC",
			"BYBIT":   "BTCUSDC",
		},
		"ETH/USDT": {
			"BINANCE": "ETHUSDT",
			"BYBIT":   "ETHUSDT",
			"OKX":     "ETH-USDT-SWAP",
			"BITGET":  "ETHUSDT",
		},
		"ETH/USDC": {
			"BINANCE": "ETHUSDC",
			"BYBIT":   "ETHUSDC",
		},
		"SOL/USDT": {
			"BINANCE": "SOLUSDT",
			"BYBIT":   "SOLUSDT",
		},
		"SOL/USDC": {
			"BINANCE": "SOLUSDC",
			"BYBIT":   "SOLUSDC",
		},
	}
	return m.LoadFromConfig(defaultConfig)
}

// GetAvailableQuotes 获取某个基础资产所有可用的结算货币
func (m *SymbolMapper) GetAvailableQuotes(baseAsset string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quotes := make(map[string]bool)
	prefix := strings.ToUpper(baseAsset) + "/"

	for canonical := range m.reverseMap {
		if strings.HasPrefix(canonical, prefix) {
			quote := strings.TrimPrefix(canonical, prefix)
			quotes[quote] = true
		}
	}

	result := make([]string, 0, len(quotes))
	for quote := range quotes {
		result = append(result, quote)
	}
	return result
}

// GetSymbolsByAssetPair 获取某个资产对在所有交易所的符号
// 例如：GetSymbolsByAssetPair("BTC", "USDT") 返回所有 BTC/USDT 的交易所符号
func (m *SymbolMapper) GetSymbolsByAssetPair(baseAsset, quoteAsset string) map[string]string {
	canonical := strings.ToUpper(baseAsset) + "/" + strings.ToUpper(quoteAsset)
	return m.GetSymbolsForCanonical(canonical)
}
