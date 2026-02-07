package port

import "context"

type Repository interface {
	// Price operations
	UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error

	// Position operations
	UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error
	GetPosition(ctx context.Context, ex, symbol string) (quantity, entryPrice float64, err error)
	ListPositions(ctx context.Context) ([]map[string]interface{}, error)

	// Snapshot operations
	InsertSnapshot(ctx context.Context, ts int64, payload string) error

	// Signal operations
	InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error

	// Connection management
	Close() error
}
