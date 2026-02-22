package okx

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

// InitializeConverter 初始化OKX的符号转换器
// 应在应用启动时调用，避免每次创建TickerFeed时都初始化
func InitializeConverter(quote string) {
	suffix := fmt.Sprintf("-%s-SWAP", quote)
	symbolConverter = exchange.NewCommonSymbolConverter(suffix)
}

type TickerFeed struct {
	wsURL string // e.g., wss://ws.okx.com:8443/ws/v5/public
}

// NewTickerFeed 创建 OKX ticker feed
func NewTickerFeed(wsURL string) *TickerFeed {
	return &TickerFeed{
		wsURL: strings.TrimSpace(wsURL),
	}
}

func (f *TickerFeed) Name() string { return application.ExchangeOKX }

// Symbol2Coin 将 OKX 格式的交易对转换为币种
func (f *TickerFeed) Symbol2Coin(symbol string) string {
	if symbolConverter == nil {
		return ""
	}
	return symbolConverter.Symbol2Coin(symbol)
}

type okxSubReq struct {
	Op   string      `json:"op"`
	Args []okxSubArg `json:"args"`
}

type okxSubArg struct {
	Channel string `json:"channel"`
	InstID  string `json:"instId"`
}

type okxTickerMsg struct {
	Op   string          `json:"op"`
	Data []okxTickerData `json:"data,omitempty"`
	Arg  okxSubArg       `json:"arg,omitempty"`
}

type okxTickerData struct {
	InstID string `json:"instId"`
	Last   string `json:"last"`
	Ts     string `json:"ts"`
}

func (f *TickerFeed) Subscribe(ctx context.Context, coins []string) (<-chan port.Tick, error) {
	if f.wsURL == "" {
		return nil, errors.New("okx wsURL empty")
	}
	if len(coins) == 0 {
		return nil, errors.New("coins empty")
	}

	out := make(chan port.Tick, 1024)
	go f.run(ctx, coins, out)
	return out, nil
}

func (f *TickerFeed) run(ctx context.Context, coins []string, out chan<- port.Tick) {
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
		// 将币种转换为 OKX 格式的交易对 (e.g., BTC -> BTC-USDT-SWAP)
		args := make([]okxSubArg, 0, len(coins))
		for _, coin := range coins {
			coin = strings.TrimSpace(coin)
			if coin == "" {
				continue
			}
			// 将币种转换为 OKX 格式的交易对 (e.g., BTC -> BTC-USDT-SWAP)
			instID := symbolConverter.Coin2Symbol(coin)
			args = append(args, okxSubArg{
				Channel: "tickers",
				InstID:  instID,
			})
			log.Debug().Str("feed", f.Name()).Str("coin", coin).Str("instID", instID).Msg("preparing subscription")
		}

		if len(args) > 0 {
			subReq := okxSubReq{
				Op:   "subscribe",
				Args: args,
			}
			if b, err := json.Marshal(subReq); err == nil {
				log.Debug().Str("feed", f.Name()).RawJSON("request", b).Msg("sending subscribe request")
				if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
					log.Error().Str("feed", f.Name()).Err(err).Msg("failed to send subscribe message")
				} else {
					log.Info().Str("feed", f.Name()).Int("count", len(args)).Msg("subscribed to tickers")
				}
			} else {
				log.Error().Str("feed", f.Name()).Err(err).Msg("failed to marshal subscribe message")
			}
		} else {
			log.Warn().Str("feed", f.Name()).Msg("no valid args for subscription")
		}

		err = readLoop(ctx, conn, func(b []byte) {
			var msg okxTickerMsg
			if e := json.Unmarshal(b, &msg); e != nil {
				log.Debug().Str("feed", f.Name()).Err(e).RawJSON("data", b).Msg("json unmarshal failed")
				return
			}

			// Log all received messages for debugging
			log.Debug().Str("feed", f.Name()).Str("op", msg.Op).Int("data_len", len(msg.Data)).Msg("received message")

			// Only process data messages with ticker info
			if len(msg.Data) == 0 {
				return
			}

			for _, data := range msg.Data {
				sym := strings.TrimSpace(data.InstID)
				pxs := strings.TrimSpace(data.Last)
				if sym == "" || pxs == "" {
					log.Debug().Str("feed", f.Name()).Str("sym", sym).Str("price", pxs).Msg("skipping empty fields")
					continue
				}

				pxn, err := strconv.ParseFloat(pxs, 64)
				if err != nil {
					log.Warn().Str("feed", f.Name()).Str("price_str", pxs).Err(err).Msg("failed to parse price")
					continue
				}

				// Parse timestamp if available
				ts := time.Now().UnixMilli()
				if data.Ts != "" {
					if tsNum, err := strconv.ParseInt(data.Ts, 10, 64); err == nil {
						ts = tsNum
					}
				}

				tick := port.Tick{
					Exchange: f.Name(),
					Symbol:   sym,
					PriceStr: pxs,
					PriceNum: pxn,
					Ts:       ts,
				}
				log.Debug().Str("feed", f.Name()).Str("symbol", sym).Float64("price", pxn).Msg("sending tick")
				out <- tick
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
		msgCount := 0
		for {
			_, b, err := conn.ReadMessage()
			if err == nil {
				_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				msgCount++
				if msgCount%100 == 1 { // Log every 100 messages to avoid spam
					log.Debug().Int("msg_count", msgCount).Int("bytes", len(b)).Msg("received raw message")
				}
			}
			if err != nil {
				log.Error().Int("total_msgs", msgCount).Err(err).Msg("readLoop error")
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
