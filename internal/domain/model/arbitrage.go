package model

import "time"

// ========== Perpetual Models ==========

// PerpetualPrice 永续合约价格
type PerpetualPrice struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Funding   float64   `json:"funding"`   // 资金费率
	NextTime  time.Time `json:"next_time"` // 下次资金费时间
	Timestamp int64     `json:"ts_ms"`
}

// SpreadArbitrage 永续合约价差套利机会
type SpreadArbitrage struct {
	ID            string  `json:"id"` // 机会唯一ID
	Symbol        string  `json:"symbol"`
	LongExchange  string  `json:"long_exchange"`  // 做多交易所
	ShortExchange string  `json:"short_exchange"` // 做空交易所
	LongPrice     float64 `json:"long_price"`
	ShortPrice    float64 `json:"short_price"`
	Spread        float64 `json:"spread"`         // 价差百分比
	SpreadAbs     float64 `json:"spread_abs"`     // 绝对价差
	ProfitPercent float64 `json:"profit_percent"` // 预期利润率（扣除费用）
	MinOrderSize  float64 `json:"min_order_size"` // 最小订单规模
	MaxOrderSize  float64 `json:"max_order_size"` // 最大订单规模
	Timestamp     int64   `json:"ts_ms"`
	ExpiresAt     int64   `json:"expires_at"` // 机会过期时间戳（毫秒）
	Confidence    float64 `json:"confidence"` // 信心度 (0-1)
}

// FundingArbitrage 永续合约资金费率套利
type FundingArbitrage struct {
	ID             string  `json:"id"` // 机会唯一ID
	Symbol         string  `json:"symbol"`
	LongExchange   string  `json:"long_exchange"`
	ShortExchange  string  `json:"short_exchange"`
	LongFunding    float64 `json:"long_funding"`
	ShortFunding   float64 `json:"short_funding"`
	FundingDiff    float64 `json:"funding_diff"`
	HoldingHours   int     `json:"holding_hours"`    // 持仓时长（小时）
	ExpectedReturn float64 `json:"expected_return"`  // 预期回报
	FundingCycleMs int64   `json:"funding_cycle_ms"` // 资金费周期（毫秒）
	NextSettleTime int64   `json:"next_settle_time"` // 下次结算时间戳
	Timestamp      int64   `json:"ts_ms"`
}

// ArbitragePosition 永续合约套利持仓
type ArbitragePosition struct {
	ID              string  `json:"id"` // 持仓ID
	Symbol          string  `json:"symbol"`
	LongExchange    string  `json:"long_exchange"`
	ShortExchange   string  `json:"short_exchange"`
	Quantity        float64 `json:"quantity"`
	LongEntryPrice  float64 `json:"long_entry_price"`
	ShortEntryPrice float64 `json:"short_entry_price"`
	EntrySpread     float64 `json:"entry_spread"`                // 开仓时的价差 %
	Status          string  `json:"status"`                      // open, closing, closed
	LongOrderID     string  `json:"long_order_id,omitempty"`     // 多头订单ID
	ShortOrderID    string  `json:"short_order_id,omitempty"`    // 空头订单ID
	TotalMargin     float64 `json:"total_margin,omitempty"`      // 占用保证金
	UnrealizedPnL   float64 `json:"unrealized_pnl,omitempty"`    // 未实现PnL
	RealizedPnL     float64 `json:"realized_pnl,omitempty"`      // 已实现PnL
	StopLossPrice   float64 `json:"stop_loss_price,omitempty"`   // 止损价差 %
	TakeProfitPrice float64 `json:"take_profit_price,omitempty"` // 止盈价差 %
	RiskLevel       string  `json:"risk_level,omitempty"`        // safe, warning, danger
	OpenTime        int64   `json:"open_time"`
	CloseTime       int64   `json:"close_time,omitempty"`
	ClosingReason   string  `json:"closing_reason,omitempty"` // 平仓原因
	Notes           string  `json:"notes,omitempty"`
}

// ========== Spot Market Models ==========

// SpotPrice 现货价格
type SpotPrice struct {
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"` // 交易对，如 BTCUSDT
	Price     float64 `json:"price"`
	Bid       float64 `json:"bid"`       // 买入价
	Ask       float64 `json:"ask"`       // 卖出价
	Volume24h float64 `json:"volume24h"` // 24小时成交量
	Available float64 `json:"available"` // 可用余额
	Timestamp int64   `json:"ts_ms"`
}

// SpotSpreadArbitrage 现货价差套利机会
type SpotSpreadArbitrage struct {
	ID            string  `json:"id"` // 机会唯一ID
	Symbol        string  `json:"symbol"`
	BuyExchange   string  `json:"buy_exchange"`  // 买入交易所
	SellExchange  string  `json:"sell_exchange"` // 卖出交易所
	BuyPrice      float64 `json:"buy_price"`
	SellPrice     float64 `json:"sell_price"`
	Spread        float64 `json:"spread"`         // 价差百分比 (卖价-买价)/买价
	SpreadAbs     float64 `json:"spread_abs"`     // 绝对价差
	ProfitPercent float64 `json:"profit_percent"` // 预期利润率（扣除费用和汇兑成本）
	ExecutionTime float64 `json:"execution_time"` // 预估执行时间（秒）
	MinOrderSize  float64 `json:"min_order_size"` // 最小订单规模
	MaxOrderSize  float64 `json:"max_order_size"` // 最大订单规模
	Timestamp     int64   `json:"ts_ms"`
	ExpiresAt     int64   `json:"expires_at"` // 机会过期时间戳（毫秒，一般10-30秒）
	Confidence    float64 `json:"confidence"` // 信心度 (0-1)
}

// SpotArbitragePosition 现货套利持仓
type SpotArbitragePosition struct {
	ID                string  `json:"id"` // 持仓ID
	Symbol            string  `json:"symbol"`
	BuyExchange       string  `json:"buy_exchange"`
	SellExchange      string  `json:"sell_exchange"`
	Quantity          float64 `json:"quantity"`                    // 持仓数量
	BuyPrice          float64 `json:"buy_price"`                   // 买入价格
	SellPrice         float64 `json:"sell_price"`                  // 卖出价格
	EntrySpread       float64 `json:"entry_spread"`                // 开仓时的价差 %
	Status            string  `json:"status"`                      // pending, buying, bought, transferring, selling, sold, cancelled, closed
	BuyOrderID        string  `json:"buy_order_id,omitempty"`      // 买入订单ID
	SellOrderID       string  `json:"sell_order_id,omitempty"`     // 卖出订单ID
	TotalCost         float64 `json:"total_cost"`                  // 总成本（含费用）
	TransferCost      float64 `json:"transfer_cost"`               // 转账成本（手续费）
	TransferFeeRate   float64 `json:"transfer_fee_rate,omitempty"` // 转账费率
	BuyFee            float64 `json:"buy_fee"`                     // 买入手续费
	SellFee           float64 `json:"sell_fee"`                    // 卖出手续费
	TransferAmount    float64 `json:"transfer_amount,omitempty"`   // 待转账金额
	UnrealizedPnL     float64 `json:"unrealized_pnl,omitempty"`    // 未实现PnL
	RealizedPnL       float64 `json:"realized_pnl,omitempty"`      // 已实现盈亏
	StopLossPrice     float64 `json:"stop_loss_price,omitempty"`   // 止损价差 %
	TakeProfitPrice   float64 `json:"take_profit_price,omitempty"` // 止盈价差 %
	RiskLevel         string  `json:"risk_level,omitempty"`        // safe, warning, danger
	BuyTime           int64   `json:"buy_time"`
	SellTime          int64   `json:"sell_time,omitempty"`
	TransferStartTime int64   `json:"transfer_start_time,omitempty"` // 转账开始时间
	CloseTime         int64   `json:"close_time,omitempty"`
	ClosingReason     string  `json:"closing_reason,omitempty"` // 平仓原因: profit_target, stop_loss, price_collapse, timeout, manual, error
	Notes             string  `json:"notes,omitempty"`
}
