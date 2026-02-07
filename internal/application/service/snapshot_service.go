package service

import (
	"context"

	"xarb/internal/application/port"
)

type SnapshotService struct {
	repo port.Repository
}

func NewSnapshotService(repo port.Repository) *SnapshotService {
	return &SnapshotService{repo: repo}
}

func (s *SnapshotService) SaveSnapshot(ctx context.Context, ts int64, payload string) error {
	return s.repo.InsertSnapshot(ctx, ts, payload)
}
