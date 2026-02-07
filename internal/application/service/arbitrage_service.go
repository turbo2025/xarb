package service

import (
	"context"
	"fmt"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/domain/model"

	"github.com/rs/zerolog/log"
)

type ArbitrageServiceImpl struct {
	repo       port.ArbitrageRepository
	calculator port.ArbitrageCalculator
	minSpread  float64 // 最小可交易价差百分比
	makerFee   float64 // 手续费率
}

func NewArbitrageService(repo port.ArbitrageRepository, calc port.ArbitrageCalculator, minSpread, makerFee float64) *ArbitrageServiceImpl {
	return &ArbitrageServiceImpl{
		repo:       repo,
		calculator: calc,
		minSpread:  minSpread,
		makerFee:   makerFee,
	}
}

// ScanSpreadOpportunities 扫描价差套利机会
func (as *ArbitrageServiceImpl) ScanSpreadOpportunities(ctx context.Context, price1, price2 *model.PerpetualPrice) error {
	if price1 == nil || price2 == nil || price1.Symbol != price2.Symbol {
		return fmt.Errorf("invalid prices for comparison")
	}

	arb := as.calculator.CalculateSpread(price1, price2, as.makerFee)
	if arb == nil {
		return nil
	}

	// 只保存有利可图的机会
	if arb.ProfitPercent > as.minSpread {
		if err := as.repo.SaveSpreadOpportunity(ctx, arb); err != nil {
			log.Error().Err(err).Str("symbol", arb.Symbol).Float64("profit", arb.ProfitPercent).Msg("save spread opportunity failed")
			return err
		}
		log.Info().
			Str("symbol", arb.Symbol).
			Str("long", arb.LongExchange).
			Str("short", arb.ShortExchange).
			Float64("spread", arb.Spread).
			Float64("profit", arb.ProfitPercent).
			Msg("spread opportunity detected")
	}

	return nil
}

// ScanFundingOpportunities 扫描资金费率套利
func (as *ArbitrageServiceImpl) ScanFundingOpportunities(ctx context.Context, price1, price2 *model.PerpetualPrice, holdingHours int) error {
	if price1 == nil || price2 == nil || price1.Symbol != price2.Symbol {
		return fmt.Errorf("invalid prices for comparison")
	}

	arb := as.calculator.CalculateFunding(price1, price2, holdingHours)
	if arb == nil {
		return nil
	}

	// 只保存预期回报 > 0 的机会
	if arb.ExpectedReturn > 0 {
		if err := as.repo.SaveFundingOpportunity(ctx, arb); err != nil {
			log.Error().Err(err).Str("symbol", arb.Symbol).Msg("save funding opportunity failed")
			return err
		}
		log.Info().
			Str("symbol", arb.Symbol).
			Float64("funding_diff", arb.FundingDiff).
			Float64("return", arb.ExpectedReturn).
			Msg("funding opportunity detected")
	}

	return nil
}

// OpenPosition 开仓
func (as *ArbitrageServiceImpl) OpenPosition(ctx context.Context, symbol, longEx, shortEx string, qty, longPrice, shortPrice float64) error {
	pos := &model.ArbitragePosition{
		ID:              fmt.Sprintf("%s_%s_%s_%d", symbol, longEx, shortEx, time.Now().UnixMilli()),
		Symbol:          symbol,
		LongExchange:    longEx,
		ShortExchange:   shortEx,
		Quantity:        qty,
		LongEntryPrice:  longPrice,
		ShortEntryPrice: shortPrice,
		EntrySpread:     (shortPrice - longPrice) / longPrice * 100,
		Status:          "open",
		OpenTime:        time.Now().UnixMilli(),
	}

	return as.repo.CreatePosition(ctx, pos)
}

// ClosePosition 平仓
func (as *ArbitrageServiceImpl) ClosePosition(ctx context.Context, posID string, longPrice, shortPrice float64) error {
	pos, err := as.repo.GetPosition(ctx, posID)
	if err != nil {
		return err
	}

	pos.Status = "closed"
	pos.CloseTime = time.Now().UnixMilli()
	pos.RealizedPnL = pos.Quantity * (shortPrice - longPrice)

	return as.repo.UpdatePosition(ctx, pos)
}

// GetPositionPnL 获取持仓盈亏
func (as *ArbitrageServiceImpl) GetPositionPnL(ctx context.Context, posID string) (float64, error) {
	pos, err := as.repo.GetPosition(ctx, posID)
	if err != nil {
		return 0, err
	}

	longPrice, err := as.repo.GetLatestPrice(ctx, pos.LongExchange, pos.Symbol)
	if err != nil {
		return 0, err
	}

	shortPrice, err := as.repo.GetLatestPrice(ctx, pos.ShortExchange, pos.Symbol)
	if err != nil {
		return 0, err
	}

	// 盈亏 = (卖出价 - 买入价) * 数量
	pnl := pos.Quantity * (shortPrice.Price - longPrice.Price)

	return pnl, nil
}
