package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/exchange"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// 包级别的符号转换器
var symbolConverter exchange.SymbolConverter

// InitializeConverter 初始化Bybit的符号转换器
// 应在应用启动时调用，避免每次创建TickerFeed时都初始化
func InitializeConverter(quote string) {
	symbolConverter = exchange.NewCommonSymbolConverter(quote)
}

type TickerFeed struct {
	wsURL string // e.g. wss://stream.bybit.com/v5/public/linear
}

// NewTickerFeed 使用自定义quote创建 Bybit ticker feed，使用包级别的转换器
func NewTickerFeed(wsURL string) *TickerFeed {
	return &TickerFeed{
		wsURL: strings.TrimSpace(wsURL),
	}
}

func (f *TickerFeed) Name() string { return application.ExchangeBybit }

type bybitSubReq struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type bybitTickerItem struct {
	Symbol    string `json:"symbol"`
	LastPrice string `json:"lastPrice"`
}

// data can be object OR array
type BybitDataList []bybitTickerItem

func (d *BybitDataList) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*d = nil
		return nil
	}
	switch b[0] {
	case '[':
		var arr []bybitTickerItem
		return json.Unmarshal(b, &arr)
	case '{':
		var one bybitTickerItem
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*d = BybitDataList{one}
		return nil
	default:
		return fmt.Errorf("unexpected data json: %s", string(b))
	}
}

func bytesTrimSpace(b []byte) []byte {
	i := 0
	j := len(b) - 1
	for i <= j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j >= i && (b[j] == ' ' || b[j] == '\n' || b[j] == '\r' || b[j] == '\t') {
		j--
	}
	if i > j {
		return []byte{}
	}
	return b[i : j+1]
}

type bybitTickerMsg struct {
	Topic string        `json:"topic"`
	Type  string        `json:"type"`
	Ts    int64         `json:"ts"`
	Data  BybitDataList `json:"data"`

	Success *bool  `json:"success,omitempty"`
	RetMsg  string `json:"ret_msg,omitempty"`
	Op      string `json:"op,omitempty"`
}

func (f *TickerFeed) Subscribe(ctx context.Context, coins []string) (<-chan port.Tick, error) {
	if f.wsURL == "" {
		return nil, errors.New("bybit ws_url empty")
	}

	// 将币种转换为 Bybit 格式的交易对 (e.g., BTC -> BTCUSDT)
	topics := make([]string, 0, len(coins))
	for _, coin := range coins {
		coin = strings.TrimSpace(coin)
		if coin == "" {
			continue
		}
		// 使用 symbolConverter 转换为交易所特定格式
		symbol := symbolConverter.Coin2Symbol(coin)
		topics = append(topics, "tickers."+strings.ToUpper(symbol))
	}

	if len(topics) == 0 {
		return nil, errors.New("no valid symbols for bybit topics")
	}

	out := make(chan port.Tick, 1024)
	go f.run(ctx, topics, out)
	return out, nil
}

func (f *TickerFeed) run(ctx context.Context, topics []string, out chan<- port.Tick) {
	defer close(out)

	backoff := 500 * time.Millisecond
	maxBackoff := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Warn().Str("feed", f.Name()).Str("url", f.wsURL).Msg("ws connecting")
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		conn, _, err := websocket.DefaultDialer.DialContext(cctx, f.wsURL, nil)
		cancel()
		if err != nil {
			log.Error().Str("feed", f.Name()).Err(err).Msg("ws dial failed")
			time.Sleep(backoff)
			backoff = minDur(backoff*2, maxBackoff)
			continue
		}

		// subscribe
		sub := bybitSubReq{Op: "subscribe", Args: topics}
		if err := conn.WriteJSON(sub); err != nil {
			_ = conn.Close()
			log.Error().Str("feed", f.Name()).Err(err).Msg("subscribe failed")
			time.Sleep(backoff)
			backoff = minDur(backoff*2, maxBackoff)
			continue
		}

		backoff = 500 * time.Millisecond
		log.Info().Str("feed", f.Name()).Msg("ws connected & subscribed")

		err = readLoop(ctx, conn, func(b []byte) {
			var msg bybitTickerMsg
			if e := json.Unmarshal(b, &msg); e != nil {
				log.Error().Str("feed", f.Name()).Err(e).Msg("json unmarshal failed")
				return
			}

			// ack
			if msg.Success != nil {
				if !*msg.Success {
					log.Error().Str("feed", f.Name()).Str("ret_msg", msg.RetMsg).Msg("subscribe not success")
				}
				return
			}

			if len(msg.Data) == 0 {
				return
			}

			for _, d := range msg.Data {
				sym := strings.ToUpper(strings.TrimSpace(d.Symbol))
				pxs := strings.TrimSpace(d.LastPrice)
				if sym == "" || pxs == "" {
					continue
				}
				pxn, _ := strconv.ParseFloat(pxs, 64)
				out <- port.Tick{
					Exchange: f.Name(),
					Symbol:   sym,
					PriceStr: pxs,
					PriceNum: pxn,
					Ts:       time.Now().UnixMilli(),
				}
			}
		})

		_ = conn.Close()

		if ctx.Err() != nil {
			return
		}

		log.Warn().Str("feed", f.Name()).Err(err).Msg("ws disconnected, reconnecting")
		time.Sleep(backoff)
		backoff = minDur(backoff*2, maxBackoff)
	}
}

func readLoop(ctx context.Context, conn *websocket.Conn, onMsg func([]byte)) error {
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for {
			_, b, err := conn.ReadMessage()
			if err == nil {
				_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			}
			if err != nil {
				errCh <- err
				return
			}
			onMsg(b)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-pingTicker.C:
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
		}
	}
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
