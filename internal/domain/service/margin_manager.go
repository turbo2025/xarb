package service

import (
	"fmt"
	"sync"
	"time"
)

// MarginManager 保证金管理器 - 控制风险敞口和保证金占用
type MarginManager struct {
	mu sync.RWMutex

	// 账户信息
	TotalMargin     float64 // 总保证金
	UsedMargin      float64 // 已使用保证金
	AvailableMargin float64 // 可用保证金
	RealizedPnL     float64 // 已实现盈亏

	// 风险控制参数
	MaxMarginPerOrder  float64 // 单笔订单最多占保证金百分比（默认 5%）
	MaxOrderCount      int     // 最多并发订单数（默认 5）
	StopLossProfitRate float64 // 利润下跌到此比例时市价平仓（默认 5%）

	// 订单追踪
	ActiveOrders map[string]*ActiveOrder      // OrderID -> OrderDetails
	Positions    map[string]*PositionTracking // Symbol -> 持仓追踪
}

// ActiveOrder 活跃订单
type ActiveOrder struct {
	OrderID        string
	Symbol         string
	Direction      string // "BUY_BINANCE_SELL_BYBIT" or "BUY_BYBIT_SELL_BINANCE"
	Quantity       float64
	MarginUsed     float64 // 占用的保证金
	ExpectedProfit float64 // 预期利润
	OrderTime      int64
	BuyOrderID     string
	SellOrderID    string
	Status         string // "PENDING", "EXECUTING", "EXECUTED", "CLOSED"
}

// PositionTracking 持仓追踪
type PositionTracking struct {
	Symbol            string
	Quantity          float64
	BuyPrice          float64
	SellPrice         float64
	EntryTime         int64
	HighestProfit     float64 // 历史最高利润
	CurrentProfit     float64 // 当前利润
	CurrentProfitRate float64 // 当前利润率 %
	CloseOrderID      string  // 平仓订单 ID
	CloseType         string  // "LIMIT" (挂单平仓) 或 "MARKET" (市价平仓)
	ClosePrice        float64 // 平仓价格
	Status            string  // "OPEN", "CLOSING", "CLOSED"
	ClosedAt          int64
}

// NewMarginManager 创建保证金管理器
func NewMarginManager(totalMargin float64) *MarginManager {
	return &MarginManager{
		TotalMargin:        totalMargin,
		UsedMargin:         0,
		AvailableMargin:    totalMargin,
		MaxMarginPerOrder:  0.05, // 单笔不超过 5%
		MaxOrderCount:      5,    // 最多 5 个并发订单
		StopLossProfitRate: 0.05, // 利润跌到 5% 市价平仓
		ActiveOrders:       make(map[string]*ActiveOrder),
		Positions:          make(map[string]*PositionTracking),
	}
}

// CanExecuteOrder 检查是否可以执行订单
func (mm *MarginManager) CanExecuteOrder(
	symbol string,
	requiredMargin float64,
) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 检查 1: 是否超过最大订单数
	if len(mm.ActiveOrders) >= mm.MaxOrderCount {
		return fmt.Errorf("max order count reached: %d/%d", len(mm.ActiveOrders), mm.MaxOrderCount)
	}

	// 检查 2: 单笔订单是否超过保证金 5%
	maxMarginForOrder := mm.TotalMargin * mm.MaxMarginPerOrder
	if requiredMargin > maxMarginForOrder {
		return fmt.Errorf("order margin %.2f USD exceeds limit %.2f USD (5%% of %.2f)",
			requiredMargin, maxMarginForOrder, mm.TotalMargin)
	}

	// 检查 3: 可用保证金是否足够
	if requiredMargin > mm.AvailableMargin {
		return fmt.Errorf("insufficient margin: need %.2f USD, have %.2f USD",
			requiredMargin, mm.AvailableMargin)
	}

	return nil
}

// RegisterOrder 注册活跃订单（执行前调用）
func (mm *MarginManager) RegisterOrder(
	orderID string,
	symbol string,
	direction string,
	quantity float64,
	marginUsed float64,
	expectedProfit float64,
) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 再次验证
	if marginUsed > mm.AvailableMargin {
		return fmt.Errorf("insufficient margin at registration time")
	}

	mm.ActiveOrders[orderID] = &ActiveOrder{
		OrderID:        orderID,
		Symbol:         symbol,
		Direction:      direction,
		Quantity:       quantity,
		MarginUsed:     marginUsed,
		ExpectedProfit: expectedProfit,
		OrderTime:      time.Now().UnixMilli(),
		Status:         "PENDING",
	}

	mm.UsedMargin += marginUsed
	mm.AvailableMargin -= marginUsed

	return nil
}

// ExecuteOrder 订单已执行（成交）
func (mm *MarginManager) ExecuteOrder(
	orderID string,
	buyOrderID string,
	sellOrderID string,
	symbol string,
	quantity float64,
	buyPrice float64,
	sellPrice float64,
) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	order, exists := mm.ActiveOrders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	// 更新订单状态
	order.Status = "EXECUTED"
	order.BuyOrderID = buyOrderID
	order.SellOrderID = sellOrderID

	// 创建持仓追踪
	expectedProfit := (sellPrice - buyPrice) * quantity
	mm.Positions[symbol] = &PositionTracking{
		Symbol:            symbol,
		Quantity:          quantity,
		BuyPrice:          buyPrice,
		SellPrice:         sellPrice,
		EntryTime:         time.Now().UnixMilli(),
		HighestProfit:     expectedProfit,
		CurrentProfit:     expectedProfit,
		CurrentProfitRate: (expectedProfit / (buyPrice * quantity)) * 100,
		Status:            "OPEN",
	}

	return nil
}

// UpdatePositionProfit 更新持仓利润（实时市价更新）
func (mm *MarginManager) UpdatePositionProfit(
	symbol string,
	currentBuyPrice float64,
	currentSellPrice float64,
) (*PositionTracking, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	pos, exists := mm.Positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found for symbol %s", symbol)
	}

	if pos.Status != "OPEN" {
		return nil, fmt.Errorf("position %s is not open", symbol)
	}

	// 重新计算实时利润
	currentProfit := (currentSellPrice - currentBuyPrice) * pos.Quantity
	currentProfitRate := (currentProfit / (currentBuyPrice * pos.Quantity)) * 100

	// 更新历史最高利润
	if currentProfit > pos.HighestProfit {
		pos.HighestProfit = currentProfit
	}

	pos.CurrentProfit = currentProfit
	pos.CurrentProfitRate = currentProfitRate

	return pos, nil
}

// NeedStopLoss 检查是否需要止损（市价平仓）
// 当利润下跌到最高利润的 5% 以下时触发
func (mm *MarginManager) NeedStopLoss(symbol string) (bool, *PositionTracking, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	pos, exists := mm.Positions[symbol]
	if !exists {
		return false, nil, fmt.Errorf("position not found for symbol %s", symbol)
	}

	if pos.Status != "OPEN" {
		return false, nil, nil
	}

	// 计算利润下跌幅度
	// 如果当前利润 < 最高利润 × (1 - 5%)
	profitDropThreshold := pos.HighestProfit * (1 - mm.StopLossProfitRate)

	if pos.CurrentProfit < profitDropThreshold {
		return true, pos, nil
	}

	return false, pos, nil
}

// ClosePositionWithLimit 挂单平仓（限价单）
func (mm *MarginManager) ClosePositionWithLimit(
	symbol string,
	limitPrice float64,
) error {
	mm.mu.Lock()
	pos, exists := mm.Positions[symbol]
	mm.mu.Unlock()

	if !exists {
		return fmt.Errorf("position not found for symbol %s", symbol)
	}

	if pos.Status != "OPEN" {
		return fmt.Errorf("position %s is not open", symbol)
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	pos.Status = "CLOSING"
	pos.CloseType = "LIMIT"
	pos.ClosePrice = limitPrice

	return nil
}

// ClosePositionWithMarket 市价平仓（确保执行）
func (mm *MarginManager) ClosePositionWithMarket(
	symbol string,
	marketPrice float64,
) error {
	mm.mu.Lock()
	pos, exists := mm.Positions[symbol]
	mm.mu.Unlock()

	if !exists {
		return fmt.Errorf("position not found for symbol %s", symbol)
	}

	if pos.Status != "OPEN" && pos.Status != "CLOSING" {
		return fmt.Errorf("position %s cannot be market closed", symbol)
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	pos.Status = "CLOSING"
	pos.CloseType = "MARKET"
	pos.ClosePrice = marketPrice

	return nil
}

// MarkPositionClosed 标记持仓已平仓
func (mm *MarginManager) MarkPositionClosed(
	symbol string,
	realizedProfit float64,
) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	pos, exists := mm.Positions[symbol]
	if !exists {
		return fmt.Errorf("position not found for symbol %s", symbol)
	}

	pos.Status = "CLOSED"
	pos.ClosedAt = time.Now().UnixMilli()

	// 释放保证金并更新已实现盈亏
	for _, order := range mm.ActiveOrders {
		if order.Symbol == symbol {
			mm.UsedMargin -= order.MarginUsed
			mm.AvailableMargin += order.MarginUsed
			mm.RealizedPnL += realizedProfit
			order.Status = "CLOSED"
			break
		}
	}

	return nil
}

// GetMarginStatus 获取保证金状态
func (mm *MarginManager) GetMarginStatus() *MarginStatus {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	usageRate := mm.UsedMargin / mm.TotalMargin

	return &MarginStatus{
		TotalMargin:       mm.TotalMargin,
		UsedMargin:        mm.UsedMargin,
		AvailableMargin:   mm.AvailableMargin,
		UsageRate:         usageRate * 100,
		ActiveOrderCount:  len(mm.ActiveOrders),
		OpenPositionCount: len(mm.Positions),
		RealizedPnL:       mm.RealizedPnL,
	}
}

// GetPositionStatus 获取持仓状态
func (mm *MarginManager) GetPositionStatus(symbol string) (*PositionTracking, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	pos, exists := mm.Positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found for symbol %s", symbol)
	}

	return pos, nil
}

// MarginStatus 保证金状态快照
type MarginStatus struct {
	TotalMargin       float64
	UsedMargin        float64
	AvailableMargin   float64
	UsageRate         float64 // 百分比
	ActiveOrderCount  int
	OpenPositionCount int
	RealizedPnL       float64
}

// CalculateRequiredMargin 计算所需保证金
// 对于合约交易，通常需要杠杆倍数的保证金
// 例如 10 倍杠杆需要 10% 的保证金
func CalculateRequiredMargin(
	entryPrice float64,
	quantity float64,
	leverage float64,
) float64 {
	// 合约保证金 = (价格 × 数量) / 杠杆
	return (entryPrice * quantity) / leverage
}

// RiskLevel 评估风险等级
func (mm *MarginManager) RiskLevel() string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	usageRate := mm.UsedMargin / mm.TotalMargin

	switch {
	case usageRate >= 0.80:
		return "CRITICAL" // 80%+ 风险极高
	case usageRate >= 0.60:
		return "HIGH" // 60-80% 风险高
	case usageRate >= 0.40:
		return "MEDIUM" // 40-60% 风险中等
	default:
		return "LOW" // < 40% 风险低
	}
}

// EstimatedLiquidationPrice 估算清算价格（如果保证金耗尽）
// 简化版：假设清算发生在保证金为 0 时
func (mm *MarginManager) EstimatedLiquidationPrice(symbol string) (float64, error) {
	mm.mu.RLock()
	pos, exists := mm.Positions[symbol]
	mm.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("position not found for symbol %s", symbol)
	}

	// 简化计算：清算价格 = 买入价 - 账户总利润 / 数量
	liquidationPrice := pos.BuyPrice - (mm.RealizedPnL / pos.Quantity)

	return liquidationPrice, nil
}
