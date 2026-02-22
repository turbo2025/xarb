package bitget

import (
	"context"
	"encoding/json"
	"errors"
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

// InitializeConverter 初始化Bitget的符号转换器
// 应在应用启动时调用，避免每次创建PerpetualTickerFeed时都初始化
func InitializeConverter(quote string) {
	symbolConverter = exchange.NewCommonSymbolConverter(quote)
}

type TickerFeed struct {
	wsURL string // e.g., wss://ws.bitget.com/spot/v1/public
}

// NewTickerFeed 创建 Bitget ticker feed
func NewTickerFeed(wsURL string) *TickerFeed {
	return &TickerFeed{
		wsURL: strings.TrimSpace(wsURL),
	}
}

func (f *TickerFeed) Name() string { return application.ExchangeBitget }

type bitgetSubReq struct {
	Op   string         `json:"op"`
	Args []bitgetSubArg `json:"args"`
}

type bitgetSubArg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstID   string `json:"instId"`
}

type bitgetTickerMsg struct {
	Action string             `json:"action"`
	Data   []bitgetTickerData `json:"data,omitempty"`
	Arg    bitgetSubArg       `json:"arg,omitempty"`
}

type bitgetTickerData struct {
	InstID string `json:"instId"`
	Last   string `json:"last"`
	Ts     string `json:"ts"`
}

func (f *TickerFeed) Subscribe(ctx context.Context, coins []string) (<-chan port.Tick, error) {
	if f.wsURL == "" {
		return nil, errors.New("bitget wsURL empty")
	}
	if len(coins) == 0 {
		return nil, errors.New("coins empty")
	}

	// 将币种转换为 Bitget 格式的交易对 (e.g., BTC -> BTCUSDT)
	symbols := make([]string, 0, len(coins))
	for _, coin := range coins {
		coin = strings.TrimSpace(coin)
		if coin == "" {
			continue
		}
		// 使用 symbolConverter 转换为交易所特定格式
		symbol := symbolConverter.Coin2Symbol(coin)
		symbols = append(symbols, symbol)
	}

	out := make(chan port.Tick, 1024)
	go f.run(ctx, symbols, out)
	return out, nil
}

func (f *TickerFeed) run(ctx context.Context, symbols []string, out chan<- port.Tick) {
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

		backoff = 500 * time.Millisecond
		log.Info().Str("feed", f.Name()).Msg("ws connected")

		// Subscribe to ticker channels
		args := make([]bitgetSubArg, 0, len(symbols))
		for _, sym := range symbols {
			sym = strings.TrimSpace(sym)
			if sym == "" {
				continue
			}
			args = append(args, bitgetSubArg{
				InstType: "SPOT",
				Channel:  "ticker",
				InstID:   sym,
			})
		}

		if len(args) > 0 {
			subReq := bitgetSubReq{
				Op:   "subscribe",
				Args: args,
			}
			if b, err := json.Marshal(subReq); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, b)
			}
		}

		err = readLoop(ctx, conn, func(b []byte) {
			var msg bitgetTickerMsg
			if e := json.Unmarshal(b, &msg); e != nil {
				log.Error().Str("feed", f.Name()).Err(e).Msg("json unmarshal failed")
				return
			}

			// Only process push action messages with ticker data
			if msg.Action != "push" || len(msg.Data) == 0 {
				return
			}

			for _, data := range msg.Data {
				sym := strings.TrimSpace(data.InstID)
				pxs := strings.TrimSpace(data.Last)
				if sym == "" || pxs == "" {
					continue
				}

				pxn, _ := strconv.ParseFloat(pxs, 64)

				// Parse timestamp if available
				ts := time.Now().UnixMilli()
				if data.Ts != "" {
					if tsNum, err := strconv.ParseInt(data.Ts, 10, 64); err == nil {
						ts = tsNum
					}
				}

				out <- port.Tick{
					Exchange: f.Name(),
					Symbol:   sym,
					PriceStr: pxs,
					PriceNum: pxn,
					Ts:       ts,
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
