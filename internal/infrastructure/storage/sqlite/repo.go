package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"xarb/internal/application/port"
)

type Repo struct {
	db *sql.DB
}

func New(path string) (*Repo, error) {
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  exchange TEXT NOT NULL,
  symbol TEXT NOT NULL,
  price REAL NOT NULL,
  ts_ms INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  UNIQUE(exchange, symbol)
);
CREATE INDEX IF NOT EXISTS idx_prices_ts ON prices(ts_ms);
CREATE INDEX IF NOT EXISTS idx_prices_symbol ON prices(symbol);

CREATE TABLE IF NOT EXISTS positions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  exchange TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity REAL NOT NULL,
  entry_price REAL NOT NULL,
  ts_ms INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE(exchange, symbol)
);
CREATE INDEX IF NOT EXISTS idx_positions_ts ON positions(ts_ms);
CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);

CREATE TABLE IF NOT EXISTS snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts_ms INTEGER NOT NULL,
  payload TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_ms);

CREATE TABLE IF NOT EXISTS signals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts_ms INTEGER NOT NULL,
  symbol TEXT NOT NULL,
  delta REAL NOT NULL,
  payload TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signals_ts ON signals(ts_ms);
CREATE INDEX IF NOT EXISTS idx_signals_symbol ON signals(symbol);
`)
	return err
}

func (r *Repo) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	now := sql.NullInt64{Int64: ts, Valid: true}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO prices(exchange, symbol, price, ts_ms, created_at) 
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(exchange, symbol) DO UPDATE SET
		price=excluded.price, ts_ms=excluded.ts_ms
	`, ex, symbol, price, ts, now.Int64)
	return err
}

func (r *Repo) UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO positions(exchange, symbol, quantity, entry_price, ts_ms, created_at, updated_at) 
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(exchange, symbol) DO UPDATE SET
		quantity=excluded.quantity, entry_price=excluded.entry_price, ts_ms=excluded.ts_ms, updated_at=excluded.updated_at
	`, ex, symbol, quantity, entryPrice, ts, ts, ts)
	return err
}

func (r *Repo) GetPosition(ctx context.Context, ex, symbol string) (quantity, entryPrice float64, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT quantity, entry_price FROM positions WHERE exchange=? AND symbol=?`, ex, symbol).
		Scan(&quantity, &entryPrice)
	return
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
	_, err := r.db.ExecContext(ctx, `INSERT INTO snapshots(ts_ms, payload, created_at) VALUES(?, ?, ?)`, ts, payload, ts)
	return err
}

func (r *Repo) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO signals(ts_ms, symbol, delta, payload, created_at) VALUES(?, ?, ?, ?, ?)`, ts, symbol, delta, payload, ts)
	return err
}

var _ port.Repository = (*Repo)(nil)
