package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// BybitSubReq is Bybit subscription request
type BybitSubReq struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// BybitTickerItem is a single ticker in Bybit response
type BybitTickerItem struct {
	Symbol    string `json:"symbol"`
	LastPrice string `json:"lastPrice"`
}

// BybitDataList can be either array or single object
type BybitDataList []BybitTickerItem

// UnmarshalJSON handles both array and object formats
func (d *BybitDataList) UnmarshalJSON(b []byte) error {
	b = BytesTrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*d = nil
		return nil
	}
	switch b[0] {
	case '[':
		var arr []BybitTickerItem
		if err := json.Unmarshal(b, &arr); err != nil {
			return err
		}
		*d = arr
		return nil
	case '{':
		var one BybitTickerItem
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*d = BybitDataList{one}
		return nil
	default:
		return fmt.Errorf("unexpected Bybit data json: %s", string(b))
	}
}

// BybitTickerMsg is Bybit ticker message
type BybitTickerMsg struct {
	Topic   string        `json:"topic"`
	Type    string        `json:"type"`
	Ts      int64         `json:"ts"`
	Data    BybitDataList `json:"data"`
	Success *bool         `json:"success,omitempty"`
	RetMsg  string        `json:"ret_msg,omitempty"`
	Op      string        `json:"op,omitempty"`
	ConnID  string        `json:"conn_id,omitempty"`
}

// Bybit exchange adapter
type Bybit struct {
	WsURL   string
	Symbols []string
}

// NewBybit creates a new Bybit adapter
func NewBybit(wsURL string, symbols []string) *Bybit {
	return &Bybit{
		WsURL:   wsURL,
		Symbols: symbols,
	}
}

// GetName returns the exchange name
func (b *Bybit) GetName() string {
	return "Bybit"
}

// Connect establishes connection to Bybit and processes messages
func (b *Bybit) Connect(ctx context.Context, handler Handler) error {
	wsURL := strings.TrimSpace(b.WsURL)
	if wsURL == "" {
		return errors.New("bybit ws_url is empty")
	}

	topics := b.buildTopics()
	if len(topics) == 0 {
		return errors.New("no valid topics to subscribe")
	}

	helper := &WSHelper{URL: wsURL}
	conn, err := helper.DialWS(ctx)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Send subscription request
	sub := BybitSubReq{Op: "subscribe", Args: topics}
	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return helper.ReadWithPing(ctx, conn, func(data []byte) {
		var msg BybitTickerMsg
		if err := ParseJSON(data, &msg); err != nil {
			// Log but continue processing
			return
		}

		// Handle subscription response
		if msg.Success != nil {
			if !*msg.Success {
				// Subscription failed, but continue
			}
			return
		}

		if len(msg.Data) == 0 {
			return
		}

		for _, d := range msg.Data {
			sym := strings.ToUpper(strings.TrimSpace(d.Symbol))
			px := strings.TrimSpace(d.LastPrice)
			if sym == "" || px == "" {
				continue
			}
			_ = handler("BYBIT", sym, px)
		}
	})
}

func (b *Bybit) buildTopics() []string {
	out := make([]string, 0, len(b.Symbols))
	for _, s := range b.Symbols {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		out = append(out, "tickers."+u)
	}
	return out
}
