package okx

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"xarb/internal/application/port"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type PerpetualTickerFeed struct {
	wsURL string // e.g., wss://ws.okx.com:8443/ws/v5/public
}

func NewPerpetualTickerFeed(wsURL string) *PerpetualTickerFeed {
	return &PerpetualTickerFeed{wsURL: strings.TrimSpace(wsURL)}
}

func (f *PerpetualTickerFeed) Name() string { return "OKX" }

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

func (f *PerpetualTickerFeed) Subscribe(ctx context.Context, symbols []string) (<-chan port.Tick, error) {
	if f.wsURL == "" {
		return nil, errors.New("okx wsURL empty")
	}
	if len(symbols) == 0 {
		return nil, errors.New("symbols empty")
	}

	out := make(chan port.Tick, 1024)
	go f.run(ctx, symbols, out)
	return out, nil
}

func (f *PerpetualTickerFeed) run(ctx context.Context, symbols []string, out chan<- port.Tick) {
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
		args := make([]okxSubArg, 0, len(symbols))
		for _, sym := range symbols {
			sym = strings.TrimSpace(sym)
			if sym == "" {
				continue
			}
			args = append(args, okxSubArg{
				Channel: "tickers",
				InstID:  sym,
			})
		}

		if len(args) > 0 {
			subReq := okxSubReq{
				Op:   "subscribe",
				Args: args,
			}
			if b, err := json.Marshal(subReq); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, b)
			}
		}

		err = readLoop(ctx, conn, func(b []byte) {
			var msg okxTickerMsg
			if e := json.Unmarshal(b, &msg); e != nil {
				log.Error().Str("feed", f.Name()).Err(e).Msg("json unmarshal failed")
				return
			}

			// Only process data messages with ticker info
			if len(msg.Data) == 0 {
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
