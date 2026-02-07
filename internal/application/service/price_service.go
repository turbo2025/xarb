package service

import (
	"context"

	"xarb/internal/application/port"
)

type PriceService struct {
	repo port.Repository
}

func NewPriceService(repo port.Repository) *PriceService {
	return &PriceService{repo: repo}
}

func (s *PriceService) UpdatePrice(ctx context.Context, exchange, symbol string, price float64, ts int64) error {
	return s.repo.UpsertLatestPrice(ctx, exchange, symbol, price, ts)
}
