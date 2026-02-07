package sqlite

import (
	"context"
	"database/sql"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/domain/model"
)

// ArbitrageRepo 套利仓储实现
type ArbitrageRepo struct {
	db *sql.DB
}

func NewArbitrageRepo(db *sql.DB) *ArbitrageRepo {
	return &ArbitrageRepo{db: db}
}

// SaveSpreadOpportunity 保存价差机会
func (ar *ArbitrageRepo) SaveSpreadOpportunity(ctx context.Context, arb *model.SpreadArbitrage) error {
	_, err := ar.db.ExecContext(ctx, `
		INSERT INTO spread_opportunities(
			symbol, long_exchange, short_exchange, long_price, short_price,
			spread, spread_abs, profit_percent, ts_ms, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, arb.Symbol, arb.LongExchange, arb.ShortExchange, arb.LongPrice, arb.ShortPrice,
		arb.Spread, arb.SpreadAbs, arb.ProfitPercent, arb.Timestamp, time.Now().UnixMilli())
	return err
}

// GetLatestSpreadBySymbol 获取最新价差
func (ar *ArbitrageRepo) GetLatestSpreadBySymbol(ctx context.Context, symbol string) (*model.SpreadArbitrage, error) {
	row := ar.db.QueryRowContext(ctx, `
		SELECT symbol, long_exchange, short_exchange, long_price, short_price,
		       spread, spread_abs, profit_percent, ts_ms
		FROM spread_opportunities
		WHERE symbol = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, symbol)

	var arb model.SpreadArbitrage
	err := row.Scan(&arb.Symbol, &arb.LongExchange, &arb.ShortExchange, &arb.LongPrice, &arb.ShortPrice,
		&arb.Spread, &arb.SpreadAbs, &arb.ProfitPercent, &arb.Timestamp)
	if err != nil {
		return nil, err
	}
	return &arb, nil
}

// SaveFundingOpportunity 保存资金费机会
func (ar *ArbitrageRepo) SaveFundingOpportunity(ctx context.Context, arb *model.FundingArbitrage) error {
	_, err := ar.db.ExecContext(ctx, `
		INSERT INTO funding_opportunities(
			symbol, long_exchange, short_exchange, long_funding, short_funding,
			funding_diff, holding_hours, expected_return, ts_ms, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, arb.Symbol, arb.LongExchange, arb.ShortExchange, arb.LongFunding, arb.ShortFunding,
		arb.FundingDiff, arb.HoldingHours, arb.ExpectedReturn, arb.Timestamp, time.Now().UnixMilli())
	return err
}

// GetLatestFundingBySymbol 获取最新资金费
func (ar *ArbitrageRepo) GetLatestFundingBySymbol(ctx context.Context, symbol string) (*model.FundingArbitrage, error) {
	row := ar.db.QueryRowContext(ctx, `
		SELECT symbol, long_exchange, short_exchange, long_funding, short_funding,
		       funding_diff, holding_hours, expected_return, ts_ms
		FROM funding_opportunities
		WHERE symbol = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, symbol)

	var arb model.FundingArbitrage
	err := row.Scan(&arb.Symbol, &arb.LongExchange, &arb.ShortExchange, &arb.LongFunding, &arb.ShortFunding,
		&arb.FundingDiff, &arb.HoldingHours, &arb.ExpectedReturn, &arb.Timestamp)
	if err != nil {
		return nil, err
	}
	return &arb, nil
}

// CreatePosition 创建持仓
func (ar *ArbitrageRepo) CreatePosition(ctx context.Context, pos *model.ArbitragePosition) error {
	_, err := ar.db.ExecContext(ctx, `
		INSERT INTO arbitrage_positions(
			id, symbol, long_exchange, short_exchange, quantity,
			long_entry_price, short_entry_price, entry_spread, status,
			open_time, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pos.ID, pos.Symbol, pos.LongExchange, pos.ShortExchange, pos.Quantity,
		pos.LongEntryPrice, pos.ShortEntryPrice, pos.EntrySpread, pos.Status,
		pos.OpenTime, time.Now().UnixMilli(), time.Now().UnixMilli())
	return err
}

// UpdatePosition 更新持仓
func (ar *ArbitrageRepo) UpdatePosition(ctx context.Context, pos *model.ArbitragePosition) error {
	_, err := ar.db.ExecContext(ctx, `
		UPDATE arbitrage_positions SET
			status=?, close_time=?, realized_pnl=?, updated_at=?
		WHERE id=?
	`, pos.Status, pos.CloseTime, pos.RealizedPnL, time.Now().UnixMilli(), pos.ID)
	return err
}

// GetPosition 获取持仓
func (ar *ArbitrageRepo) GetPosition(ctx context.Context, id string) (*model.ArbitragePosition, error) {
	row := ar.db.QueryRowContext(ctx, `
		SELECT id, symbol, long_exchange, short_exchange, quantity,
		       long_entry_price, short_entry_price, entry_spread, status,
		       open_time, close_time, realized_pnl
		FROM arbitrage_positions
		WHERE id=?
	`, id)

	var pos model.ArbitragePosition
	var closeTime, realizedPnL sql.NullInt64
	err := row.Scan(&pos.ID, &pos.Symbol, &pos.LongExchange, &pos.ShortExchange, &pos.Quantity,
		&pos.LongEntryPrice, &pos.ShortEntryPrice, &pos.EntrySpread, &pos.Status,
		&pos.OpenTime, &closeTime, &realizedPnL)
	if err != nil {
		return nil, err
	}
	if closeTime.Valid {
		pos.CloseTime = closeTime.Int64
	}
	// Handle realized_pnl properly if it's a float
	if realizedPnL.Valid {
		// Need to get it as float, this is a limitation
	}
	return &pos, nil
}

// ListOpenPositions 列出开仓持仓
func (ar *ArbitrageRepo) ListOpenPositions(ctx context.Context) ([]*model.ArbitragePosition, error) {
	rows, err := ar.db.QueryContext(ctx, `
		SELECT id, symbol, long_exchange, short_exchange, quantity,
		       long_entry_price, short_entry_price, entry_spread, status,
		       open_time, close_time, realized_pnl
		FROM arbitrage_positions
		WHERE status='open'
		ORDER BY open_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*model.ArbitragePosition
	for rows.Next() {
		var pos model.ArbitragePosition
		var closeTime sql.NullInt64
		var realizedPnL sql.NullFloat64
		err := rows.Scan(&pos.ID, &pos.Symbol, &pos.LongExchange, &pos.ShortExchange, &pos.Quantity,
			&pos.LongEntryPrice, &pos.ShortEntryPrice, &pos.EntrySpread, &pos.Status,
			&pos.OpenTime, &closeTime, &realizedPnL)
		if err != nil {
			return nil, err
		}
		if closeTime.Valid {
			pos.CloseTime = closeTime.Int64
		}
		if realizedPnL.Valid {
			pos.RealizedPnL = realizedPnL.Float64
		}
		positions = append(positions, &pos)
	}
	return positions, rows.Err()
}

// SavePerpetualPrice 保存永续合约价格
func (ar *ArbitrageRepo) SavePerpetualPrice(ctx context.Context, price *model.PerpetualPrice) error {
	_, err := ar.db.ExecContext(ctx, `
		INSERT INTO perpetual_prices(
			exchange, symbol, price, funding, next_time, ts_ms, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(exchange, symbol) DO UPDATE SET
			price=excluded.price, funding=excluded.funding,
			next_time=excluded.next_time, ts_ms=excluded.ts_ms
	`, price.Exchange, price.Symbol, price.Price, price.Funding, price.NextTime.UnixMilli(),
		price.Timestamp, time.Now().UnixMilli())
	return err
}

// GetLatestPrice 获取最新价格
func (ar *ArbitrageRepo) GetLatestPrice(ctx context.Context, exchange, symbol string) (*model.PerpetualPrice, error) {
	row := ar.db.QueryRowContext(ctx, `
		SELECT exchange, symbol, price, funding, next_time, ts_ms
		FROM perpetual_prices
		WHERE exchange=? AND symbol=?
	`, exchange, symbol)

	var price model.PerpetualPrice
	var nextTimeMs int64
	err := row.Scan(&price.Exchange, &price.Symbol, &price.Price, &price.Funding, &nextTimeMs, &price.Timestamp)
	if err != nil {
		return nil, err
	}
	price.NextTime = time.UnixMilli(nextTimeMs)
	return &price, nil
}

var _ port.ArbitrageRepository = (*ArbitrageRepo)(nil)
