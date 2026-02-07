package monitor

import (
	"context"

	"xarb/internal/application/port"
)

type noopRepo struct{}

func NewNoopRepo() port.Repository { return &noopRepo{} }

func (n *noopRepo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	return nil
}
func (n *noopRepo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	return nil
}
func (n *noopRepo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	return nil
}

func (n *noopRepo) UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error {
	return nil
}

func (n *noopRepo) GetPosition(ctx context.Context, ex, symbol string) (float64, float64, error) {
	return 0, 0, nil
}

func (n *noopRepo) ListPositions(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}

func (n *noopRepo) Close() error {
	return nil
}
