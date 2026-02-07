package service

import (
	"context"

	"xarb/internal/application/port"
)

type PositionService struct {
	repo port.Repository
}

func NewPositionService(repo port.Repository) *PositionService {
	return &PositionService{repo: repo}
}

func (s *PositionService) UpdatePosition(ctx context.Context, exchange, symbol string, quantity, entryPrice float64, ts int64) error {
	return s.repo.UpsertPosition(ctx, exchange, symbol, quantity, entryPrice, ts)
}

func (s *PositionService) GetPosition(ctx context.Context, exchange, symbol string) (quantity, entryPrice float64, err error) {
	return s.repo.GetPosition(ctx, exchange, symbol)
}

func (s *PositionService) ListAllPositions(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.ListPositions(ctx)
}
