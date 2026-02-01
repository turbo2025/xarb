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
	b pxState
	y pxState
}

type State struct {
	mu sync.Mutex

	order []string
	syms  map[string]*symState
}

func NewState(symbols []string) *State {
	order := make([]string, 0, len(symbols))
	syms := make(map[string]*symState, len(symbols))
	for _, s := range symbols {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		order = append(order, u)
		syms[u] = &symState{}
	}
	return &State{order: order, syms: syms}
}

func (s *State) Symbols() []string {
	return s.order
}

// Apply returns true if the displayed price string changed (so we should redraw)
func (s *State) Apply(t port.Tick) bool {
	ex := strings.ToUpper(strings.TrimSpace(t.Exchange))
	sym := strings.ToUpper(strings.TrimSpace(t.Symbol))
	price := strings.TrimSpace(t.PriceStr)
	if sym == "" || price == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.syms[sym]
	if st == nil {
		return false
	}

	var ps *pxState
	switch ex {
	case "BINANCE":
		ps = &st.b
	case "BYBIT":
		ps = &st.y
	default:
		return false
	}

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
