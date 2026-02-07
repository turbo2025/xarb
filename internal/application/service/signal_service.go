package service

import (
	"context"

	"xarb/internal/application/port"
)

type SignalService struct {
	repo port.Repository
}

func NewSignalService(repo port.Repository) *SignalService {
	return &SignalService{repo: repo}
}

func (s *SignalService) CreateSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	return s.repo.InsertSignal(ctx, ts, symbol, delta, payload)
}
