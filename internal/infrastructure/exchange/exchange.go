package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Handler receives messages from the exchange
type Handler func(exchange, symbol, price string) error

// Exchange defines the interface for exchange adapters
type Exchange interface {
	Connect(ctx context.Context, handler Handler) error
	GetName() string
}

// WSHelper provides common WebSocket functionality
type WSHelper struct {
	URL string
}

// DialWS creates a WebSocket connection with timeout
func (w *WSHelper) DialWS(ctx context.Context) (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, w.URL, nil)
	return conn, err
}

// ReadWithPing reads WebSocket messages with periodic pings
func (w *WSHelper) ReadWithPing(ctx context.Context, conn *websocket.Conn, onMessage func([]byte)) error {
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
			if err != nil {
				errCh <- err
				return
			}
			onMessage(b)
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

// MinDuration returns the minimum of two durations
func MinDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// Runner provides base reconnection logic for exchanges
type Runner struct {
	Exchange Exchange
	Logger   Logger
}

// Logger interface for logging
type Logger interface {
	Error(msg string, err error)
	Warn(msg string)
	Info(msg string)
	Debug(msg string)
}

// Run executes the exchange runner with automatic reconnection
func (r *Runner) Run(ctx context.Context, handler Handler) {
	backoff := 500 * time.Millisecond
	maxBackoff := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r.Logger.Warn(fmt.Sprintf("%s connecting", r.Exchange.GetName()))

		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := r.Exchange.Connect(cctx, handler)
		cancel()

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		if err != nil {
			r.Logger.Warn(fmt.Sprintf("%s disconnected: %v, reconnecting", r.Exchange.GetName(), err))
		}

		time.Sleep(backoff)
		backoff = MinDuration(backoff*2, maxBackoff)
	}
}

// BytesTrimSpace trims whitespace from byte slice
func BytesTrimSpace(b []byte) []byte {
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

// ParseJSON safely parses JSON
func ParseJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// BuildQueryURL builds a URL with query parameters
func BuildQueryURL(base, path, query string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", errors.New("base url is empty")
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path
	u.RawQuery = query
	return u.String(), nil
}
