package port

import "context"

type Repository interface {
	UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error
	InsertSnapshot(ctx context.Context, ts int64, payload string) error
	InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error
}
