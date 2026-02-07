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
CREATE TABLE IF NOT EXISTS prices (
  id BIGSERIAL PRIMARY KEY,
  exchange VARCHAR(50) NOT NULL,
  symbol VARCHAR(50) NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  ts_ms BIGINT NOT NULL,
  created_at BIGINT NOT NULL,
  UNIQUE(exchange, symbol)
);
CREATE INDEX IF NOT EXISTS idx_prices_ts ON prices(ts_ms);
CREATE INDEX IF NOT EXISTS idx_prices_symbol ON prices(symbol);

CREATE TABLE IF NOT EXISTS positions (
  id BIGSERIAL PRIMARY KEY,
  exchange VARCHAR(50) NOT NULL,
  symbol VARCHAR(50) NOT NULL,
  quantity DOUBLE PRECISION NOT NULL,
  entry_price DOUBLE PRECISION NOT NULL,
  ts_ms BIGINT NOT NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  UNIQUE(exchange, symbol)
);
CREATE INDEX IF NOT EXISTS idx_positions_ts ON positions(ts_ms);
CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);

CREATE TABLE IF NOT EXISTS snapshots (
  id BIGSERIAL PRIMARY KEY,
  ts_ms BIGINT NOT NULL,
  payload TEXT NOT NULL,
  created_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_ms);

CREATE TABLE IF NOT EXISTS signals (
  id BIGSERIAL PRIMARY KEY,
  ts_ms BIGINT NOT NULL,
  symbol VARCHAR(50) NOT NULL,
  delta DOUBLE PRECISION NOT NULL,
  payload TEXT NOT NULL,
  created_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signals_ts ON signals(ts_ms);
CREATE INDEX IF NOT EXISTS idx_signals_symbol ON signals(symbol);
`)
	return err
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO prices(exchange, symbol, price, ts_ms, created_at)
		VALUES($1, $2, $3, $4, $5)
		ON CONFLICT(exchange, symbol) DO UPDATE SET
		price=excluded.price, ts_ms=excluded.ts_ms
	`, ex, symbol, price, ts, ts)
	return err
}

func (r *Repo) UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO positions(exchange, symbol, quantity, entry_price, ts_ms, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(exchange, symbol) DO UPDATE SET
		quantity=excluded.quantity, entry_price=excluded.entry_price, ts_ms=excluded.ts_ms, updated_at=excluded.updated_at
	`, ex, symbol, quantity, entryPrice, ts, ts, ts)
	return err
}

func (r *Repo) GetPosition(ctx context.Context, ex, symbol string) (float64, float64, error) {
	var quantity, entryPrice float64
	err := r.db.QueryRowContext(ctx, `SELECT quantity, entry_price FROM positions WHERE exchange=$1 AND symbol=$2`, ex, symbol).
		Scan(&quantity, &entryPrice)
	return quantity, entryPrice, err
}

func (r *Repo) ListPositions(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT exchange, symbol, quantity, entry_price, ts_ms FROM positions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []map[string]interface{}
	for rows.Next() {
		var exchange, symbol string
		var quantity, entryPrice float64
		var ts int64
		if err := rows.Scan(&exchange, &symbol, &quantity, &entryPrice, &ts); err != nil {
			return nil, err
		}
		positions = append(positions, map[string]interface{}{
			"exchange":   exchange,
			"symbol":     symbol,
			"quantity":   quantity,
			"entryPrice": entryPrice,
			"ts_ms":      ts,
		})
	}
	return positions, rows.Err()
}

func (r *Repo) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO snapshots(ts_ms, payload, created_at) VALUES($1, $2, $3)`, ts, payload, ts)
	return err
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO signals(ts_ms, symbol, delta, payload, created_at) VALUES($1, $2, $3, $4, $5)`, ts, symbol, delta, payload, ts)
	return err
}

var _ port.Repository = (*Repo)(nil)
