package service

import (
	"fmt"
	"sync"

	"xarb/internal/domain/model"
)

// RiskManager 风险管理器
type RiskManager struct {
	mu sync.RWMutex

	// 头寸限制
	MaxPositionSizeUSD    float64 // 单个头寸最大美元值
	MaxTotalExposureUSD   float64 // 总敞口最大美元值
	MaxPositionsPerSymbol int     // 单个符号最多头寸数
	MaxTotalPositions     int     // 最多总头寸数

	// 相关性阈值
	CorrelationThreshold float64 // 相关性限制（避免高度相关头寸）

	// 当前头寸跟踪
	positions map[string][]*model.ArbitragePosition // symbol -> positions
	symbols   map[string]float64                    // symbol -> last price
}

// NewRiskManager 创建风险管理器
func NewRiskManager() *RiskManager {
	return &RiskManager{
		MaxPositionSizeUSD:    100000,  // 单个头寸最多 10 万美元
		MaxTotalExposureUSD:   1000000, // 总敞口最多 100 万美元
		MaxPositionsPerSymbol: 3,       // 单个符号最多3个头寸
		MaxTotalPositions:     10,      // 最多10个头寸
		CorrelationThreshold:  0.8,     // 相关性 > 0.8 时限制
		positions:             make(map[string][]*model.ArbitragePosition),
		symbols:               make(map[string]float64),
	}
}

// CanOpenPosition 检查是否可以开仓
func (rm *RiskManager) CanOpenPosition(opportunity *model.SpreadArbitrage, quantity float64) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 检查单个头寸大小
	positionValue := opportunity.LongPrice * quantity
	if positionValue > rm.MaxPositionSizeUSD {
		return fmt.Errorf("position size %.2f USD exceeds limit %.2f USD",
			positionValue, rm.MaxPositionSizeUSD)
	}

	// 检查单个符号头寸数
	if len(rm.positions[opportunity.Symbol]) >= rm.MaxPositionsPerSymbol {
		return fmt.Errorf("symbol %s already has %d positions (max %d)",
			opportunity.Symbol, len(rm.positions[opportunity.Symbol]), rm.MaxPositionsPerSymbol)
	}

	// 检查总头寸数
	totalPositions := rm.countTotalPositions()
	if totalPositions >= rm.MaxTotalPositions {
		return fmt.Errorf("total positions %d exceeds max %d",
			totalPositions, rm.MaxTotalPositions)
	}

	// 检查总敞口
	totalExposure := rm.calculateTotalExposure()
	newExposure := totalExposure + positionValue
	if newExposure > rm.MaxTotalExposureUSD {
		return fmt.Errorf("new exposure %.2f USD exceeds limit %.2f USD",
			newExposure, rm.MaxTotalExposureUSD)
	}

	// 检查相关性（可选：与现有头寸的相关性）
	if err := rm.checkCorrelation(opportunity.Symbol); err != nil {
		return err
	}

	return nil
}

// RegisterPosition 注册新头寸
func (rm *RiskManager) RegisterPosition(pos *model.ArbitragePosition) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.positions[pos.Symbol] == nil {
		rm.positions[pos.Symbol] = make([]*model.ArbitragePosition, 0)
	}
	rm.positions[pos.Symbol] = append(rm.positions[pos.Symbol], pos)
}

// ClosePosition 关闭头寸
func (rm *RiskManager) ClosePosition(posID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for symbol, positions := range rm.positions {
		for i, pos := range positions {
			if pos.ID == posID {
				// 从切片移除
				rm.positions[symbol] = append(positions[:i], positions[i+1:]...)
				if len(rm.positions[symbol]) == 0 {
					delete(rm.positions, symbol)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("position %s not found", posID)
}

// UpdatePrice 更新符号价格
func (rm *RiskManager) UpdatePrice(symbol string, price float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.symbols[symbol] = price
}

// CalculateRiskMetrics 计算风险指标
func (rm *RiskManager) CalculateRiskMetrics() *RiskMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	metrics := &RiskMetrics{
		TotalPositions:    rm.countTotalPositions(),
		TotalExposureUSD:  rm.calculateTotalExposure(),
		PositionsBySymbol: make(map[string]int),
	}

	for symbol, positions := range rm.positions {
		metrics.PositionsBySymbol[symbol] = len(positions)
	}

	return metrics
}

// countTotalPositions 计算总头寸数
func (rm *RiskManager) countTotalPositions() int {
	total := 0
	for _, positions := range rm.positions {
		total += len(positions)
	}
	return total
}

// calculateTotalExposure 计算总敞口
func (rm *RiskManager) calculateTotalExposure() float64 {
	total := 0.0
	for symbol, positions := range rm.positions {
		price := rm.symbols[symbol]
		if price <= 0 {
			price = 1.0 // 默认值
		}
		for _, pos := range positions {
			total += pos.Quantity * price
		}
	}
	return total
}

// checkCorrelation 检查相关性（简化实现）
func (rm *RiskManager) checkCorrelation(newSymbol string) error {
	// 实现简单的相关性检查：同一大类资产（如都是BTC）不能有过多头寸
	// 这里是一个占位符实现
	btcLike := []string{"BTCUSDT", "BTCUSD", "BTC-USDT"}

	isBTC := false
	for _, s := range btcLike {
		if newSymbol == s {
			isBTC = true
			break
		}
	}

	if isBTC {
		btcCount := 0
		for sym := range rm.positions {
			for _, s := range btcLike {
				if sym == s {
					btcCount++
				}
			}
		}
		if btcCount >= 2 {
			return fmt.Errorf("BTC-like correlations too high, already have %d BTC positions", btcCount)
		}
	}

	return nil
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	TotalPositions    int
	TotalExposureUSD  float64
	PositionsBySymbol map[string]int
	VaR95             float64 // 95% Value at Risk
	MaxDrawdown       float64
}

// ValidatePosition 验证头寸有效性
func (rm *RiskManager) ValidatePosition(pos *model.ArbitragePosition) error {
	if pos.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if pos.LongEntryPrice <= 0 || pos.ShortEntryPrice <= 0 {
		return fmt.Errorf("entry prices must be positive")
	}
	if pos.LongEntryPrice == pos.ShortEntryPrice {
		return fmt.Errorf("long and short entry prices cannot be equal")
	}
	if pos.LongExchange == pos.ShortExchange {
		return fmt.Errorf("long and short exchanges must be different")
	}
	return nil
}

// CalculatePNL 计算头寸 PnL
func (rm *RiskManager) CalculatePNL(pos *model.ArbitragePosition, longPrice, shortPrice float64) float64 {
	longPnL := (longPrice - pos.LongEntryPrice) * pos.Quantity
	shortPnL := (pos.ShortEntryPrice - shortPrice) * pos.Quantity
	return longPnL + shortPnL
}

// CalculatePNLPercent 计算 PnL 百分比
func (rm *RiskManager) CalculatePNLPercent(pos *model.ArbitragePosition, longPrice, shortPrice float64) float64 {
	if pos.LongEntryPrice <= 0 {
		return 0
	}
	pnl := rm.CalculatePNL(pos, longPrice, shortPrice)
	capital := (pos.LongEntryPrice + pos.ShortEntryPrice) / 2 * pos.Quantity
	if capital == 0 {
		return 0
	}
	return (pnl / capital) * 100
}

// SuggestClosePrice 建议平仓价格
func (rm *RiskManager) SuggestClosePrice(pos *model.ArbitragePosition, targetProfitPercent float64) (float64, float64) {
	// 长仓平仓价格：入场价 * (1 + 目标利润%)
	longClosePrice := pos.LongEntryPrice * (1 + targetProfitPercent/100)
	// 短仓平仓价格：入场价 * (1 - 目标利润%)
	shortClosePrice := pos.ShortEntryPrice * (1 - targetProfitPercent/100)
	return longClosePrice, shortClosePrice
}

// CalculateStopLoss 计算止损价格
func (rm *RiskManager) CalculateStopLoss(pos *model.ArbitragePosition, stopLossPercent float64) (float64, float64) {
	// 长仓止损：入场价 * (1 - 止损%)
	longStopLoss := pos.LongEntryPrice * (1 - stopLossPercent/100)
	// 短仓止损：入场价 * (1 + 止损%)
	shortStopLoss := pos.ShortEntryPrice * (1 + stopLossPercent/100)
	return longStopLoss, shortStopLoss
}

// CalculateExpectedPnL 计算预期 PnL
func (rm *RiskManager) CalculateExpectedPnL(spread float64, quantity float64, entryPrice float64, feePercent float64) float64 {
	// 预期 PnL = 差价 * 数量 - 手续费
	grossPnL := spread * quantity * entryPrice / 100     // 假设差价是百分比
	fees := entryPrice * quantity * feePercent / 100 * 2 // 买卖双向手续费
	return grossPnL - fees
}

// IsHealthy 检查风险管理器健康状态
func (rm *RiskManager) IsHealthy() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	totalPos := rm.countTotalPositions()
	if totalPos > rm.MaxTotalPositions {
		return false
	}

	totalExposure := rm.calculateTotalExposure()
	if totalExposure > rm.MaxTotalExposureUSD {
		return false
	}

	for _, positions := range rm.positions {
		if len(positions) > rm.MaxPositionsPerSymbol {
			return false
		}
	}

	return true
}

// SetLimits 设置限制参数
func (rm *RiskManager) SetLimits(maxPos, maxExposure float64, maxPerSymbol, maxTotal int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if maxPos > 0 {
		rm.MaxPositionSizeUSD = maxPos
	}
	if maxExposure > 0 {
		rm.MaxTotalExposureUSD = maxExposure
	}
	if maxPerSymbol > 0 {
		rm.MaxPositionsPerSymbol = maxPerSymbol
	}
	if maxTotal > 0 {
		rm.MaxTotalPositions = maxTotal
	}
}
