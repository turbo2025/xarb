package postgres

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"xarb/internal/application/port"
)

type Repo struct {
	db *sql.DB
}

func New(dsn string) (*Repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	r := &Repo{db: db}
	if err := r.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return r, nil
}

func (r *Repo) Close() error { return r.db.Close() }

func (r *Repo) migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS snapshots (
  id BIGSERIAL PRIMARY KEY,
  ts_ms BIGINT NOT NULL,
  payload TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_ms);
`)
	return err
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	// optional: add latest table later
	return nil
}

func (r *Repo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO snapshots(ts_ms, payload) VALUES($1, $2)`, ts, payload)
	return err
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	return nil
}

var _ port.Repository = (*Repo)(nil)
