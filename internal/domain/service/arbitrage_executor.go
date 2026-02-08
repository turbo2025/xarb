package service

import (
	"fmt"
)

// ArbitrageExecutor 套利执行引擎 - 根据价差和成本计算纯利润，决定是否下单
type ArbitrageExecutor struct {
	// 交易费用（万分比）
	BinanceMakerFee float64 // Binance 挂单手续费，默认 0.02%
	BinanceTakerFee float64 // Binance 吃单手续费，默认 0.04%
	BybitMakerFee   float64 // Bybit 挂单手续费，默认 0.01%
	BybitTakerFee   float64 // Bybit 吃单手续费，默认 0.03%

	// 融资费率（小时费率，需要转换为每笔成本）
	BinanceFundingRate float64 // Binance 融资费率（例如 0.001 = 0.1%）
	BybitFundingRate   float64 // Bybit 融资费率

	// 利润阈值
	MinProfitPercentage float64 // 最小利润率，默认 0.1% (万分比)

	// 持仓时间（小时，用于计算融资费）
	HoldingHours float64 // 默认 1 小时

	// 订单执行模式
	MakerTakerMode bool // true: 挂单模式，false: 吃单模式
}

// NewArbitrageExecutor 创建套利执行器
func NewArbitrageExecutor() *ArbitrageExecutor {
	return &ArbitrageExecutor{
		// 默认费用设置
		BinanceMakerFee: 0.02, // 0.02%
		BinanceTakerFee: 0.04, // 0.04%
		BybitMakerFee:   0.01, // 0.01%
		BybitTakerFee:   0.03, // 0.03%

		// 融资费率（典型值）
		BinanceFundingRate: 0.001,  // 0.1% 每 8 小时
		BybitFundingRate:   0.0008, // 0.08% 每 8 小时

		// 利润要求
		MinProfitPercentage: 0.1, // 最少 0.1% 纯利润

		// 持仓时间
		HoldingHours: 1,

		// 默认吃单模式（更快成交）
		MakerTakerMode: false,
	}
}

// OpportunityAnalysis 套利机会分析结果
type OpportunityAnalysis struct {
	Symbol           string
	BinancePrice     float64
	BybitPrice       float64
	Spread           float64 // 价差
	SpreadPercentage float64 // 价差百分比

	// 成本分析
	TotalTradingFeeRate float64 // 总交易费率
	FundingFeeRate      float64 // 融资费率
	TotalCostRate       float64 // 总成本比例

	// 利润分析
	GrossProfitRate float64 // 毛利率（仅考虑价差）
	NetProfitRate   float64 // 净利率（考虑所有成本）
	NetProfitUSD    float64 // 净利润（美元）

	// 决策
	IsOpportunity bool   // 是否是套利机会
	Reason        string // 决策原因
}

// AnalyzeOpportunity 分析套利机会
// direction: "BUY_BINANCE_SELL_BYBIT" 或 "BUY_BYBIT_SELL_BINANCE"
func (ae *ArbitrageExecutor) AnalyzeOpportunity(
	symbol string,
	binancePrice float64,
	bybitPrice float64,
	quantity float64,
) *OpportunityAnalysis {
	result := &OpportunityAnalysis{
		Symbol:       symbol,
		BinancePrice: binancePrice,
		BybitPrice:   bybitPrice,
		Spread:       bybitPrice - binancePrice,
	}

	// 价差百分比 = (Bybit - Binance) / Binance
	if binancePrice > 0 {
		result.SpreadPercentage = (result.Spread / binancePrice) * 100 // 百分比
	}

	// 策略1: 在 Binance 买入，在 Bybit 卖出（Bybit 价格高）
	// 费用: Binance 吃单 + Bybit 挂单 + 融资费
	if bybitPrice > binancePrice {
		result.TotalTradingFeeRate = ae.BinanceTakerFee + ae.BybitMakerFee
		result.FundingFeeRate = (ae.BinanceFundingRate + ae.BybitFundingRate) * ae.HoldingHours / 8
		result.GrossProfitRate = result.SpreadPercentage
		result.TotalCostRate = result.TotalTradingFeeRate + result.FundingFeeRate*100

		result.NetProfitRate = result.GrossProfitRate - result.TotalCostRate
		result.NetProfitUSD = (binancePrice * quantity) * (result.NetProfitRate / 100)

		// 判断是否有利可图
		if result.NetProfitRate >= ae.MinProfitPercentage {
			result.IsOpportunity = true
			result.Reason = fmt.Sprintf(
				"BUY_BINANCE_SELL_BYBIT: Spread=%.4f%%, Cost=%.4f%%, NetProfit=%.4f%%, USD=%.2f",
				result.SpreadPercentage, result.TotalCostRate, result.NetProfitRate, result.NetProfitUSD,
			)
		} else {
			result.Reason = fmt.Sprintf(
				"Spread too small: %.4f%% < MinProfit %.4f%% + Cost %.4f%%",
				result.SpreadPercentage, ae.MinProfitPercentage, result.TotalCostRate,
			)
		}
		return result
	}

	// 策略2: 在 Bybit 买入，在 Binance 卖出（Binance 价格高）
	// 费用: Bybit 吃单 + Binance 挂单 + 融资费
	if binancePrice > bybitPrice {
		reverseSpread := binancePrice - bybitPrice
		result.Spread = -reverseSpread
		result.SpreadPercentage = (reverseSpread / bybitPrice) * 100

		result.TotalTradingFeeRate = ae.BybitTakerFee + ae.BinanceMakerFee
		result.FundingFeeRate = (ae.BinanceFundingRate + ae.BybitFundingRate) * ae.HoldingHours / 8
		result.GrossProfitRate = result.SpreadPercentage
		result.TotalCostRate = result.TotalTradingFeeRate + result.FundingFeeRate*100

		result.NetProfitRate = result.GrossProfitRate - result.TotalCostRate
		result.NetProfitUSD = (bybitPrice * quantity) * (result.NetProfitRate / 100)

		if result.NetProfitRate >= ae.MinProfitPercentage {
			result.IsOpportunity = true
			result.Reason = fmt.Sprintf(
				"BUY_BYBIT_SELL_BINANCE: Spread=%.4f%%, Cost=%.4f%%, NetProfit=%.4f%%, USD=%.2f",
				result.SpreadPercentage, result.TotalCostRate, result.NetProfitRate, result.NetProfitUSD,
			)
		} else {
			result.Reason = fmt.Sprintf(
				"Spread too small: %.4f%% < MinProfit %.4f%% + Cost %.4f%%",
				result.SpreadPercentage, ae.MinProfitPercentage, result.TotalCostRate,
			)
		}
		return result
	}

	// 价格相同
	result.Reason = "No spread: prices are equal"
	return result
}

// CalculateOrderDetails 计算订单详情
func (ae *ArbitrageExecutor) CalculateOrderDetails(
	symbol string,
	binancePrice float64,
	bybitPrice float64,
	quantity float64,
) (*OrderDetails, error) {
	// 分析机会
	analysis := ae.AnalyzeOpportunity(symbol, binancePrice, bybitPrice, quantity)

	if !analysis.IsOpportunity {
		return nil, fmt.Errorf("not profitable: %s", analysis.Reason)
	}

	// 计算订单详情
	orders := &OrderDetails{
		Symbol:         symbol,
		Quantity:       quantity,
		Spread:         analysis.Spread,
		NetProfitRate:  analysis.NetProfitRate,
		ExpectedProfit: analysis.NetProfitUSD,
	}

	// 确定交易方向
	if bybitPrice > binancePrice {
		orders.Direction = "BUY_BINANCE_SELL_BYBIT"
		orders.BuyPrice = binancePrice
		orders.SellPrice = bybitPrice
		orders.BuyCost = binancePrice * quantity * (1 + ae.BinanceTakerFee/100)
		orders.SellRevenue = bybitPrice * quantity * (1 - ae.BybitMakerFee/100)
	} else {
		orders.Direction = "BUY_BYBIT_SELL_BINANCE"
		orders.BuyPrice = bybitPrice
		orders.SellPrice = binancePrice
		orders.BuyCost = bybitPrice * quantity * (1 + ae.BybitTakerFee/100)
		orders.SellRevenue = binancePrice * quantity * (1 - ae.BinanceMakerFee/100)
	}

	orders.NetProfit = orders.SellRevenue - orders.BuyCost

	return orders, nil
}

// OrderDetails 订单详情
type OrderDetails struct {
	Symbol         string
	Direction      string // "BUY_BINANCE_SELL_BYBIT" or "BUY_BYBIT_SELL_BINANCE"
	Quantity       float64
	Spread         float64
	BuyPrice       float64
	SellPrice      float64
	BuyCost        float64 // 包括费用
	SellRevenue    float64 // 扣除费用
	NetProfit      float64 // 实际利润
	NetProfitRate  float64 // 利润率 %
	ExpectedProfit float64

	// 订单状态跟踪
	BuyOrderID  string
	SellOrderID string
	ExecutedAt  int64 // Unix milliseconds
}

// SetFees 自定义设置费用
func (ae *ArbitrageExecutor) SetFees(
	binanceMaker, binanceTaker, bybitMaker, bybitTaker float64,
) {
	ae.BinanceMakerFee = binanceMaker
	ae.BinanceTakerFee = binanceTaker
	ae.BybitMakerFee = bybitMaker
	ae.BybitTakerFee = bybitTaker
}

// SetFundingRates 自定义设置融资费率
func (ae *ArbitrageExecutor) SetFundingRates(binance, bybit float64) {
	ae.BinanceFundingRate = binance
	ae.BybitFundingRate = bybit
}

// SetMinProfitThreshold 设置最小利润阈值
func (ae *ArbitrageExecutor) SetMinProfitThreshold(percentage float64) {
	ae.MinProfitPercentage = percentage
}

// SetHoldingHours 设置预计持仓时间
func (ae *ArbitrageExecutor) SetHoldingHours(hours float64) {
	ae.HoldingHours = hours
}
