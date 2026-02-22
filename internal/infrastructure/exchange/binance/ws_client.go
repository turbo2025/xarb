package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

// InitializeConverter 初始化Binance的符号转换器
// 应在应用启动时调用，避免每次创建PerpetualTickerFeed时都初始化
func InitializeConverter(quote string) {
	symbolConverter = exchange.NewCommonSymbolConverter(quote)
}

type TickerFeed struct {
	wsURL string // e.g. wss://fstream.binance.com
}

// NewTickerFeed 使用自定义quote创建 Binance ticker feed，使用包级别的转换器
func NewTickerFeed(wsURL string) *TickerFeed {
	return &TickerFeed{
		wsURL: strings.TrimSpace(wsURL),
	}
}

func (f *TickerFeed) Name() string { return application.ExchangeBinance }

// Symbol2Coin 将 Binance 格式的交易对转换为币种
func (f *TickerFeed) Symbol2Coin(symbol string) string {
	if symbolConverter == nil {
		return ""
	}
	return symbolConverter.Symbol2Coin(symbol)
}

type binanceCombined struct {
	Stream string         `json:"stream"`
	Data   binanceMiniMsg `json:"data"`
}
type binanceMiniMsg struct {
	Symbol string `json:"s"`
	Close  string `json:"c"`
}

func (f *TickerFeed) Subscribe(ctx context.Context, coins []string) (<-chan port.Tick, error) {
	// 将币种转换为 Binance 格式的交易对 (e.g., BTC -> BTCUSDT)
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

	wsURL, err := buildCombinedURL(f.wsURL, symbols)
	if err != nil {
		return nil, err
	}

	out := make(chan port.Tick, 1024)
	go f.run(ctx, wsURL, out)
	return out, nil
}

func buildCombinedURL(base string, symbols []string) (string, error) {
	if base == "" {
		return "", errors.New("binance ws_base empty")
	}
	if len(symbols) == 0 {
		return "", errors.New("symbols empty")
	}

	streams := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		streams = append(streams, fmt.Sprintf("%s@miniTicker", s))
	}
	if len(streams) == 0 {
		return "", errors.New("no valid symbols")
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = "/stream"
	u.RawQuery = "streams=" + strings.Join(streams, "/")
	return u.String(), nil
}

func (f *TickerFeed) run(ctx context.Context, wsURL string, out chan<- port.Tick) {
	defer close(out)

	backoff := 500 * time.Millisecond
	maxBackoff := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Warn().Str("feed", f.Name()).Str("url", wsURL).Msg("ws connecting")
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		conn, _, err := websocket.DefaultDialer.DialContext(cctx, wsURL, nil)
		cancel()
		if err != nil {
			log.Error().Str("feed", f.Name()).Err(err).Msg("ws dial failed")
			time.Sleep(backoff)
			backoff = minDur(backoff*2, maxBackoff)
			continue
		}

		backoff = 500 * time.Millisecond
		log.Info().Str("feed", f.Name()).Msg("ws connected")

		err = readLoop(ctx, conn, func(b []byte) {
			var msg binanceCombined
			if e := json.Unmarshal(b, &msg); e != nil {
				log.Error().Str("feed", f.Name()).Err(e).Msg("json unmarshal failed")
				return
			}
			sym := strings.ToUpper(msg.Data.Symbol)
			pxs := strings.TrimSpace(msg.Data.Close)
			if sym == "" || pxs == "" {
				return
			}
			pxn, _ := strconv.ParseFloat(pxs, 64)
			out <- port.Tick{
				Exchange: f.Name(),
				Symbol:   sym,
				PriceStr: pxs,
				PriceNum: pxn,
				Ts:       time.Now().UnixMilli(),
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
