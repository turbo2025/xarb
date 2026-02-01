package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// BinanceCombined wraps Binance combined stream message
type BinanceCombined struct {
	Stream string         `json:"stream"`
	Data   BinanceMiniMsg `json:"data"`
}

// BinanceMiniMsg is Binance miniTicker message
type BinanceMiniMsg struct {
	Symbol string `json:"s"`
	Close  string `json:"c"`
}

// Binance exchange adapter
type Binance struct {
	BaseURL string
	Symbols []string
}

// NewBinance creates a new Binance adapter
func NewBinance(baseURL string, symbols []string) *Binance {
	return &Binance{
		BaseURL: baseURL,
		Symbols: symbols,
	}
}

// GetName returns the exchange name
func (b *Binance) GetName() string {
	return "Binance"
}

// Connect establishes connection to Binance and processes messages
func (b *Binance) Connect(ctx context.Context, handler Handler) error {
	wsURL, err := b.buildURL()
	if err != nil {
		return fmt.Errorf("build url: %w", err)
	}

	helper := &WSHelper{URL: wsURL}
	conn, err := helper.DialWS(ctx)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	return helper.ReadWithPing(ctx, conn, func(data []byte) {
		var msg BinanceCombined
		if err := ParseJSON(data, &msg); err != nil {
			// Log but continue processing
			return
		}

		sym := strings.ToUpper(msg.Data.Symbol)
		px := msg.Data.Close
		if sym == "" || px == "" {
			return
		}

		_ = handler("BINANCE", sym, px)
	})
}

func (b *Binance) buildURL() (string, error) {
	if b.BaseURL == "" {
		return "", errors.New("binance ws_url is empty")
	}
	if len(b.Symbols) == 0 {
		return "", errors.New("symbols list is empty")
	}

	streams := make([]string, 0, len(b.Symbols))
	for _, s := range b.Symbols {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		streams = append(streams, fmt.Sprintf("%s@miniTicker", s))
	}
	if len(streams) == 0 {
		return "", errors.New("no valid symbols found")
	}

	return BuildQueryURL(b.BaseURL, "/stream", "streams="+strings.Join(streams, "/"))
}
