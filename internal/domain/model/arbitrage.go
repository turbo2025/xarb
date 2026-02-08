package model

import "time"

// FuturesPrice 永续合约价格
type FuturesPrice struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Funding   float64   `json:"funding"`   // 资金费率
	NextTime  time.Time `json:"next_time"` // 下次资金费时间
	Timestamp int64     `json:"ts_ms"`
}

// SpreadArbitrage 价差套利机会
type SpreadArbitrage struct {
	Symbol        string  `json:"symbol"`
	LongExchange  string  `json:"long_exchange"`  // 做多交易所
	ShortExchange string  `json:"short_exchange"` // 做空交易所
	LongPrice     float64 `json:"long_price"`
	ShortPrice    float64 `json:"short_price"`
	Spread        float64 `json:"spread"`         // 价差百分比
	SpreadAbs     float64 `json:"spread_abs"`     // 绝对价差
	ProfitPercent float64 `json:"profit_percent"` // 预期利润率（扣除费用）
	Timestamp     int64   `json:"ts_ms"`
}

// FundingArbitrage 资金费率套利
type FundingArbitrage struct {
	Symbol         string  `json:"symbol"`
	LongExchange   string  `json:"long_exchange"`
	ShortExchange  string  `json:"short_exchange"`
	LongFunding    float64 `json:"long_funding"`
	ShortFunding   float64 `json:"short_funding"`
	FundingDiff    float64 `json:"funding_diff"`
	HoldingHours   int     `json:"holding_hours"`   // 持仓时长（小时）
	ExpectedReturn float64 `json:"expected_return"` // 预期回报
	Timestamp      int64   `json:"ts_ms"`
}

// ArbitragePosition 套利持仓
type ArbitragePosition struct {
	ID              string  `json:"id"`
	Symbol          string  `json:"symbol"`
	LongExchange    string  `json:"long_exchange"`
	ShortExchange   string  `json:"short_exchange"`
	Quantity        float64 `json:"quantity"`
	LongEntryPrice  float64 `json:"long_entry_price"`
	ShortEntryPrice float64 `json:"short_entry_price"`
	EntrySpread     float64 `json:"entry_spread"`
	Status          string  `json:"status"` // open, closing, closed
	OpenTime        int64   `json:"open_time"`
	CloseTime       int64   `json:"close_time,omitempty"`
	RealizedPnL     float64 `json:"realized_pnl,omitempty"`
}
