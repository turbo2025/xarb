package service

import (
	"testing"
)

func TestSymbolMapper_RegisterMultiQuote(t *testing.T) {
	mapper := NewSymbolMapper()

	// 为 BTC 注册多个结算货币
	err := mapper.RegisterMultiQuote("BINANCE", "BTC", []string{"USDT", "USDC"})
	if err != nil {
		t.Errorf("RegisterMultiQuote failed: %v", err)
	}

	// 验证 BTC/USDT
	canonical, err := mapper.ToCanonical("BINANCE", "BTCUSDT")
	if err != nil || canonical != "BTC/USDT" {
		t.Errorf("expected BTC/USDT, got %s", canonical)
	}

	// 验证 BTC/USDC
	canonical, err = mapper.ToCanonical("BINANCE", "BTCUSDC")
	if err != nil || canonical != "BTC/USDC" {
		t.Errorf("expected BTC/USDC, got %s", canonical)
	}
}

func TestSymbolMapper_GetAvailableQuotes(t *testing.T) {
	mapper := NewSymbolMapper()
	mapper.RegisterMultiQuote("BINANCE", "BTC", []string{"USDT", "USDC", "BUSD"})

	quotes := mapper.GetAvailableQuotes("BTC")
	if len(quotes) != 3 {
		t.Errorf("expected 3 quotes, got %d", len(quotes))
	}

	// 检查所有结算货币都在
	quoteMap := make(map[string]bool)
	for _, q := range quotes {
		quoteMap[q] = true
	}

	expectedQuotes := []string{"USDT", "USDC", "BUSD"}
	for _, expected := range expectedQuotes {
		if !quoteMap[expected] {
			t.Errorf("quote %s not found", expected)
		}
	}
}

func TestSymbolMapper_GetSymbolsByAssetPair(t *testing.T) {
	mapper := NewSymbolMapper()
	mapper.Register("BINANCE", "BTCUSDT", "BTC/USDT")
	mapper.Register("BYBIT", "BTCUSDT", "BTC/USDT")
	mapper.Register("OKX", "BTC-USDT-SWAP", "BTC/USDT")

	symbols := mapper.GetSymbolsByAssetPair("BTC", "USDT")
	if len(symbols) != 3 {
		t.Errorf("expected 3 exchanges, got %d", len(symbols))
	}

	if symbols["BINANCE"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT for BINANCE, got %s", symbols["BINANCE"])
	}
	if symbols["OKX"] != "BTC-USDT-SWAP" {
		t.Errorf("expected BTC-USDT-SWAP for OKX, got %s", symbols["OKX"])
	}
}

func TestSymbolMapper_LoadDefaultConfig(t *testing.T) {
	mapper := NewSymbolMapper()
	err := mapper.LoadDefaultConfig()
	if err != nil {
		t.Errorf("LoadDefaultConfig failed: %v", err)
	}

	// 验证 BTC/USDT 在 Binance 上
	canonical, err := mapper.ToCanonical("BINANCE", "BTCUSDT")
	if err != nil || canonical != "BTC/USDT" {
		t.Errorf("expected BTC/USDT, got %s", canonical)
	}

	// 验证 ETH/USDC 在 Bybit 上
	canonical, err = mapper.ToCanonical("BYBIT", "ETHUSDC")
	if err != nil || canonical != "ETH/USDC" {
		t.Errorf("expected ETH/USDC, got %s", canonical)
	}

	// 验证 SOL/USDT 在 Binance 上
	canonical, err = mapper.ToCanonical("BINANCE", "SOLUSDT")
	if err != nil || canonical != "SOL/USDT" {
		t.Errorf("expected SOL/USDT, got %s", canonical)
	}

	// 验证 SOL 有两个结算货币
	quotes := mapper.GetAvailableQuotes("SOL")
	if len(quotes) != 2 {
		t.Errorf("expected 2 SOL quotes, got %d: %v", len(quotes), quotes)
	}
}

func TestSymbolMapper_RegisterMultiExchange(t *testing.T) {
	mapper := NewSymbolMapper()

	// 在多个交易所注册相同符号
	err := mapper.RegisterMultiExchange(
		[]string{"BINANCE", "BYBIT", "OKX"},
		"BTCUSDT",
		"BTC/USDT",
	)
	if err != nil {
		t.Errorf("RegisterMultiExchange failed: %v", err)
	}

	// 验证三个交易所都有映射
	for _, exchange := range []string{"BINANCE", "BYBIT", "OKX"} {
		canonical, err := mapper.ToCanonical(exchange, "BTCUSDT")
		if err != nil || canonical != "BTC/USDT" {
			t.Errorf("expected BTC/USDT for %s, got error: %v or value: %s", exchange, err, canonical)
		}
	}
}
