package composite

import (
	"context"

	"xarb/internal/application/port"
)

type Repo struct {
	repos []port.Repository
}

func New(repos ...port.Repository) *Repo {
	// nil repos are allowed; filter in constructor for safety
	out := make([]port.Repository, 0, len(repos))
	for _, r := range repos {
		if r != nil {
			out = append(out, r)
		}
	}
	return &Repo{repos: out}
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	var firstErr error
	for _, repo := range r.repos {
		if err := repo.UpsertLatestPrice(ctx, ex, symbol, price, ts); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *Repo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	var firstErr error
	for _, repo := range r.repos {
		if err := repo.InsertSnapshot(ctx, ts, payload); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	var firstErr error
	for _, repo := range r.repos {
		if err := repo.InsertSignal(ctx, ts, symbol, delta, payload); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
