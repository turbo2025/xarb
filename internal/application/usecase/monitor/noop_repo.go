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
