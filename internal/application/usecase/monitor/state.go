package monitor

import (
	"strconv"
	"strings"
	"sync"
	"xarb/internal/application/port"
)

type Dir int

const (
	DirSame Dir = 0
	DirUp   Dir = +1
	DirDown Dir = -1
)

type pxState struct {
	str   string
	num   float64
	has   bool
	dir   Dir
	seen  bool
	parse bool
}

type symState struct {
	exchanges map[string]*pxState // exchange -> price state (e.g., "BINANCE" -> *pxState)
}

type State struct {
	mu sync.Mutex

	order []string
	syms  map[string]*symState
}

func NewState(coins []string) *State {
	order := make([]string, 0, len(coins))
	syms := make(map[string]*symState, len(coins))
	for _, coin := range coins {
		u := strings.ToUpper(strings.TrimSpace(coin))
		if u == "" {
			continue
		}
		order = append(order, u)
		syms[u] = &symState{
			exchanges: make(map[string]*pxState),
		}
	}
	return &State{order: order, syms: syms}
}

func (s *State) Symbols() []string {
	return s.order
}

// Apply 应用一个币种的价格更新，返回是否显示更新（相对于前一个价格发生了变化）
// Tick 中的 Symbol 应该是币种名称（如 "BTC", "ETH"），而不是交易对格式
func (s *State) Apply(t port.Tick) bool {
	ex := strings.ToUpper(strings.TrimSpace(t.Exchange))
	coin := strings.ToUpper(strings.TrimSpace(t.Symbol)) // Symbol 应该是币种，非交易对
	price := strings.TrimSpace(t.PriceStr)
	if coin == "" || price == "" || ex == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.syms[coin]
	if st == nil {
		return false
	}

	// 获取或创建该交易所的价格状态
	ps := st.exchanges[ex]
	if ps == nil {
		ps = &pxState{}
		st.exchanges[ex] = ps
	}

	// 检查价格是否变化
	if ps.str == price {
		ps.seen = true
		return false
	}

	ps.str = price
	ps.seen = true

	n, err := strconv.ParseFloat(price, 64)
	if err != nil {
		ps.parse = false
		ps.dir = DirSame
		return true
	}

	ps.parse = true
	if !ps.has {
		ps.has = true
		ps.num = n
		ps.dir = DirSame
		return true
	}

	prev := ps.num
	switch {
	case n > prev:
		ps.dir = DirUp
	case n < prev:
		ps.dir = DirDown
	default:
		ps.dir = DirSame
	}
	ps.num = n
	return true
}

func (s *State) Snapshot() map[string]symState {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]symState, len(s.syms))
	for k, v := range s.syms {
		out[k] = *v
	}
	return out
}

// DeltaBandFor 计算两个特定交易所间的价差和带状分级
func (s *State) DeltaBandFor(symbol string, ex1, ex2 string, threshold float64) (delta float64, band int, ok bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	e1 := strings.ToUpper(strings.TrimSpace(ex1))
	e2 := strings.ToUpper(strings.TrimSpace(ex2))

	if sym == "" || e1 == "" || e2 == "" || threshold <= 0 {
		return 0, 0, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.syms[sym]
	if st == nil {
		return 0, 0, false
	}

	// 获取两个交易所的价格状态
	ps1 := st.exchanges[e1]
	ps2 := st.exchanges[e2]

	// 两边都必须 parse & has 才能算 delta
	if ps1 == nil || ps2 == nil || !(ps1.parse && ps2.parse && ps1.has && ps2.has) {
		return 0, 0, false
	}

	d := ps2.num - ps1.num
	switch {
	case d >= threshold:
		return d, +1, true
	case d <= -threshold:
		return d, -1, true
	default:
		return d, 0, true
	}
}

// DeltaBand 对所有交易所对进行两两比较，返回最大和最小的 delta（已弃用，建议使用 DeltaBandFor）
func (s *State) DeltaBand(symbol string, threshold float64) (delta float64, band int, ok bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" || threshold <= 0 {
		return 0, 0, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.syms[sym]
	if st == nil {
		return 0, 0, false
	}

	if len(st.exchanges) < 2 {
		return 0, 0, false
	}

	// 获取所有交易所（按字母序，确保一致性）
	var exchanges []string
	for ex := range st.exchanges {
		exchanges = append(exchanges, ex)
	}
	if len(exchanges) < 2 {
		return 0, 0, false
	}

	// 简化版本：使用前两个交易所（按字母序）
	// 更好的做法是在 Service 层配置要比较的交易所
	ps1 := st.exchanges[exchanges[0]]
	ps2 := st.exchanges[exchanges[1]]

	if ps1 == nil || ps2 == nil || !(ps1.parse && ps2.parse && ps1.has && ps2.has) {
		return 0, 0, false
	}

	d := ps2.num - ps1.num
	switch {
	case d >= threshold:
		return d, +1, true
	case d <= -threshold:
		return d, -1, true
	default:
		return d, 0, true
	}
}

// GetExchangePrices 获取某个交易对的所有交易所价格（用于两两比较）
func (s *State) GetExchangePrices(symbol string) map[string]*pxState {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.syms[sym]
	if st == nil {
		return nil
	}

	// 返回副本，避免外部修改
	out := make(map[string]*pxState, len(st.exchanges))
	for ex, ps := range st.exchanges {
		out[ex] = ps
	}
	return out
}

// GetExchangeNames 获取某个交易对下所有有效交易所的名称列表
func (s *State) GetExchangeNames(symbol string) []string {
	prices := s.GetExchangePrices(symbol)
	var names []string
	for ex := range prices {
		names = append(names, ex)
	}
	return names
}
