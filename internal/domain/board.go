package domain

import (
	"strings"
	"sync"
)

// SymbolState holds the price state for both exchanges
type SymbolState struct {
	Binance *PriceState
	Bybit   *PriceState
}

// Board manages price tracking for multiple symbols across exchanges
type Board struct {
	mu        sync.RWMutex
	order     []string                // ordered symbol list
	symbols   map[string]*SymbolState // symbol -> state mapping
	threshold float64                 // delta threshold for alerting
}

// NewBoard creates a new Board instance
func NewBoard(symbols []string, threshold float64) *Board {
	order := make([]string, 0, len(symbols))
	syms := make(map[string]*SymbolState, len(symbols))

	for _, s := range symbols {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		order = append(order, u)
		syms[u] = &SymbolState{
			Binance: &PriceState{},
			Bybit:   &PriceState{},
		}
	}

	return &Board{
		order:     order,
		symbols:   syms,
		threshold: threshold,
	}
}

// Update updates the price for a symbol on a specific exchange
// Returns true if the price has changed
func (b *Board) Update(exchange, symbol, price string) bool {
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	price = strings.TrimSpace(price)

	if exchange != "BINANCE" && exchange != "BYBIT" && exchange != "B" && exchange != "Y" {
		return false
	}
	if symbol == "" || price == "" {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	st := b.symbols[symbol]
	if st == nil {
		return false
	}

	var ps *PriceState
	if exchange == "BINANCE" || exchange == "B" {
		ps = st.Binance
	} else {
		ps = st.Bybit
	}

	return ps.Update(price)
}

// GetSnapshot returns a read-only snapshot of current state
func (b *Board) GetSnapshot() map[string]*SymbolState {
	b.mu.RLock()
	defer b.mu.RUnlock()

	snap := make(map[string]*SymbolState, len(b.symbols))
	for sym, state := range b.symbols {
		snap[sym] = &SymbolState{
			Binance: &PriceState{
				String:    state.Binance.String,
				Number:    state.Binance.Number,
				HasValue:  state.Binance.HasValue,
				Direction: state.Binance.Direction,
				IsSeen:    state.Binance.IsSeen,
				IsParsed:  state.Binance.IsParsed,
			},
			Bybit: &PriceState{
				String:    state.Bybit.String,
				Number:    state.Bybit.Number,
				HasValue:  state.Bybit.HasValue,
				Direction: state.Bybit.Direction,
				IsSeen:    state.Bybit.IsSeen,
				IsParsed:  state.Bybit.IsParsed,
			},
		}
	}
	return snap
}

// GetSymbols returns ordered list of symbols
func (b *Board) GetSymbols() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]string, len(b.order))
	copy(result, b.order)
	return result
}

// GetThreshold returns the delta threshold
func (b *Board) GetThreshold() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.threshold
}
