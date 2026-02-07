package service

import (
	"xarb/internal/domain/model"
)

type ArbitrageCalculator struct {
	makerFeePercentage float64 // 默认 maker 手续费率
}

func NewArbitrageCalculator(makerFeePercentage float64) *ArbitrageCalculator {
	return &ArbitrageCalculator{
		makerFeePercentage: makerFeePercentage,
	}
}

// CalculateSpread 计算价差套利机会
func (ac *ArbitrageCalculator) CalculateSpread(long *model.PerpetualPrice, short *model.PerpetualPrice, makerFee float64) *model.SpreadArbitrage {
	if long == nil || short == nil {
		return nil
	}

	// 价差（绝对值）
	spreadAbs := long.Price - short.Price

	// 价差百分比
	spreadPercent := (spreadAbs / short.Price) * 100

	// 手续费成本（往返）
	feeCost := (long.Price * makerFee) + (short.Price * makerFee)

	// 预期利润率 = 价差 - 手续费
	profitPercent := spreadPercent - (feeCost / short.Price * 100)

	return &model.SpreadArbitrage{
		Symbol:        long.Symbol,
		LongExchange:  short.Exchange, // 在低价交易所做多
		ShortExchange: long.Exchange,  // 在高价交易所做空
		LongPrice:     short.Price,
		ShortPrice:    long.Price,
		Spread:        spreadPercent,
		SpreadAbs:     spreadAbs,
		ProfitPercent: profitPercent,
		Timestamp:     long.Timestamp,
	}
}

// CalculateFunding 计算资金费率套利
func (ac *ArbitrageCalculator) CalculateFunding(long *model.PerpetualPrice, short *model.PerpetualPrice, holdingHours int) *model.FundingArbitrage {
	if long == nil || short == nil {
		return nil
	}

	fundingDiff := long.Funding - short.Funding

	// 假设8小时结算一次（Binance），计算预期回报
	fundingCycles := holdingHours / 8
	expectedReturn := fundingDiff * 100 * float64(fundingCycles)

	return &model.FundingArbitrage{
		Symbol:         long.Symbol,
		LongExchange:   long.Exchange,
		ShortExchange:  short.Exchange,
		LongFunding:    long.Funding,
		ShortFunding:   short.Funding,
		FundingDiff:    fundingDiff,
		HoldingHours:   holdingHours,
		ExpectedReturn: expectedReturn,
		Timestamp:      long.Timestamp,
	}
}
