package port

import (
	"context"

	"xarb/internal/domain/model"
)

// ArbitrageRepository 套利数据仓储
type ArbitrageRepository interface {
	// Spread arbitrage
	SaveSpreadOpportunity(ctx context.Context, arb *model.SpreadArbitrage) error
	GetLatestSpreadBySymbol(ctx context.Context, symbol string) (*model.SpreadArbitrage, error)

	// Funding arbitrage
	SaveFundingOpportunity(ctx context.Context, arb *model.FundingArbitrage) error
	GetLatestFundingBySymbol(ctx context.Context, symbol string) (*model.FundingArbitrage, error)

	// Positions
	CreatePosition(ctx context.Context, pos *model.ArbitragePosition) error
	UpdatePosition(ctx context.Context, pos *model.ArbitragePosition) error
	GetPosition(ctx context.Context, id string) (*model.ArbitragePosition, error)
	ListOpenPositions(ctx context.Context) ([]*model.ArbitragePosition, error)

	// Perpetual prices
	SavePerpetualPrice(ctx context.Context, price *model.PerpetualPrice) error
	GetLatestPrice(ctx context.Context, exchange, symbol string) (*model.PerpetualPrice, error)
}

// ArbitrageCalculator 套利计算器
type ArbitrageCalculator interface {
	CalculateSpread(long *model.PerpetualPrice, short *model.PerpetualPrice, makerFee float64) *model.SpreadArbitrage
	CalculateFunding(long *model.PerpetualPrice, short *model.PerpetualPrice, holdingHours int) *model.FundingArbitrage
}
