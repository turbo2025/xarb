package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"xarb/internal/application/port"

	"github.com/redis/go-redis/v9"
)

type Repo struct {
	client        *redis.Client
	prefix        string
	ttl           time.Duration
	signalStream  string
	signalChannel string
}

func New(client *redis.Client, prefix string, ttl time.Duration, signalStream, signalChannel string) *Repo {
	return &Repo{
		client:        client,
		prefix:        prefix,
		ttl:           ttl,
		signalStream:  signalStream,
		signalChannel: signalChannel,
	}
}

func (r *Repo) key(parts ...string) string {
	fullKey := r.prefix
	for _, part := range parts {
		fullKey += ":" + part
	}
	return fullKey
}

func (r *Repo) Close() error {
	return r.client.Close()
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	key := r.key("price", ex, symbol)
	data := map[string]interface{}{"price": price, "ts_ms": ts}
	b, _ := json.Marshal(data)
	return r.client.Set(ctx, key, b, r.ttl).Err()
}

func (r *Repo) UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error {
	key := r.key("position", ex, symbol)
	data := map[string]interface{}{"quantity": quantity, "entryPrice": entryPrice, "ts_ms": ts}
	b, _ := json.Marshal(data)
	return r.client.Set(ctx, key, b, r.ttl).Err()
}

func (r *Repo) GetPosition(ctx context.Context, ex, symbol string) (float64, float64, error) {
	key := r.key("position", ex, symbol)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	var data map[string]float64
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return 0, 0, err
	}
	return data["quantity"], data["entryPrice"], nil
}

func (r *Repo) ListPositions(ctx context.Context) ([]map[string]interface{}, error) {
	pattern := r.key("position", "*", "*")
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var positions []map[string]interface{}
	for _, key := range keys {
		val, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			continue
		}
		positions = append(positions, data)
	}
	return positions, nil
}

func (r *Repo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	key := r.key("snapshot", fmt.Sprintf("%d", ts))
	return r.client.Set(ctx, key, payload, r.ttl).Err()
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	key := r.key("signal", symbol, fmt.Sprintf("%d", ts))
	data := map[string]interface{}{"delta": delta, "payload": payload}
	b, _ := json.Marshal(data)
	return r.client.Set(ctx, key, b, r.ttl).Err()
}

var _ port.Repository = (*Repo)(nil)
