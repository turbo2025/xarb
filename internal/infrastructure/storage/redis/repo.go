package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"xarb/internal/application/port"

	"github.com/redis/go-redis/v9"
)

type Repo struct {
	rdb          *redis.Client
	prefix       string
	ttl          time.Duration
	keyLatest    string // prefix + ":latest"
	signalStream string
	signalChan   string
}

type LatestPrice struct {
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Ts       int64   `json:"ts"`
}

func New(rdb *redis.Client, prefix string, ttl time.Duration, signalStream, signalChan string) *Repo {
	if strings.TrimSpace(signalStream) == "" {
		signalStream = prefix + ":signals"
	}
	if strings.TrimSpace(signalChan) == "" {
		signalChan = prefix + ":signals:pub"
	}
	return &Repo{
		rdb:          rdb,
		prefix:       prefix,
		ttl:          ttl,
		keyLatest:    prefix + ":latest",
		signalStream: signalStream,
		signalChan:   signalChan,
	}
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	if price <= 0 {
		return nil
	}
	lp := LatestPrice{Exchange: ex, Symbol: symbol, Price: price, Ts: ts}
	b, _ := json.Marshal(lp)

	// Hash: field = "BINANCE:BTCUSDT" -> json
	field := fmt.Sprintf("%s:%s", ex, symbol)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, r.keyLatest, field, string(b))
	if r.ttl > 0 {
		pipe.Expire(ctx, r.keyLatest, r.ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	// optional: store snapshots in Redis stream / list later
	return nil
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	// 1) Stream: XADD <stream> * ts symbol delta payload
	_, err := r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: r.signalStream,
		Values: map[string]any{
			"ts_ms":   ts,
			"symbol":  symbol,
			"delta":   delta,
			"payload": payload,
		},
	}).Result()
	if err != nil {
		return err
	}

	// 2) PubSub: PUBLISH <channel> json
	// 用最简单的 JSON，便于消费者
	msg := fmt.Sprintf(`{"ts_ms":%d,"symbol":"%s","delta":%.8f,"payload":%q}`, ts, symbol, delta, payload)
	return r.rdb.Publish(ctx, r.signalChan, msg).Err()
}

var _ port.Repository = (*Repo)(nil)
