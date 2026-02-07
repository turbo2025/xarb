package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OrderDeduplicator 订单去重器 - 防止重复下单
type OrderDeduplicator struct {
	mu sync.RWMutex

	// 最近的订单记录
	recentOrders map[string]*RecentOrder // symbol -> RecentOrder

	// 配置参数
	DeduplicationWindow time.Duration // 去重时间窗口（默认 5 秒）
	CooldownPeriod      time.Duration // 冷却期（同一对下一次下单需要等待）
	MaxOrdersPerSymbol  int           // 单个交易对最多待成交订单数
	OrderStateCheckTime time.Duration // 订单状态验证周期
}

// RecentOrder 最近的订单记录
type RecentOrder struct {
	OrderID     string
	Symbol      string
	Direction   string // "BUY_BINANCE_SELL_BYBIT" 等
	Quantity    float64
	BuyPrice    float64
	SellPrice   float64
	PlacedAt    int64  // Unix milliseconds
	BuyOrderID  string // Binance 订单 ID
	SellOrderID string // Bybit 订单 ID
	Status      string // "PENDING", "EXECUTING", "EXECUTED", "FAILED"
	FailReason  string // 失败原因
	ExecutedAt  int64  // 成交时间
	LastCheckAt int64  // 上次状态检查时间
}

// NewOrderDeduplicator 创建订单去重器
func NewOrderDeduplicator() *OrderDeduplicator {
	return &OrderDeduplicator{
		recentOrders:        make(map[string]*RecentOrder),
		DeduplicationWindow: 5 * time.Second,  // 5 秒内不重复下单
		CooldownPeriod:      10 * time.Second, // 同一对需要冷却 10 秒
		MaxOrdersPerSymbol:  2,                // 单对最多 2 个待成交订单
		OrderStateCheckTime: 30 * time.Second, // 每 30 秒检查一次状态
	}
}

// CanPlaceOrder 检查是否可以下单（去重检查）
func (od *OrderDeduplicator) CanPlaceOrder(
	symbol string,
	direction string,
	quantity float64,
) (bool, string) {
	od.mu.Lock()
	defer od.mu.Unlock()

	// 检查 1: 时间窗口内是否有相同方向的订单
	recent, exists := od.recentOrders[symbol]
	if exists && recent.Status != "EXECUTED" && recent.Status != "FAILED" {
		timeSinceLast := time.Since(time.UnixMilli(recent.PlacedAt))

		// 如果距离上次下单不足 5 秒，且方向相同，拒绝
		if timeSinceLast < od.DeduplicationWindow && recent.Direction == direction {
			return false, fmt.Sprintf(
				"same order within deduplication window (%.1fs ago, direction: %s)",
				timeSinceLast.Seconds(), direction)
		}

		// 如果距离上次成交不足冷却期，拒绝
		if recent.Status == "EXECUTED" && timeSinceLast < od.CooldownPeriod {
			return false, fmt.Sprintf(
				"cooldown period not met (%.1fs remaining)",
				(od.CooldownPeriod - timeSinceLast).Seconds())
		}

		// 如果有待成交的订单数过多，拒绝
		pendingCount := od.countPendingOrders(symbol)
		if pendingCount >= od.MaxOrdersPerSymbol {
			return false, fmt.Sprintf(
				"too many pending orders: %d/%d", pendingCount, od.MaxOrdersPerSymbol)
		}
	}

	return true, ""
}

// RegisterOrder 注册新订单
func (od *OrderDeduplicator) RegisterOrder(
	symbol string,
	direction string,
	quantity float64,
	buyPrice float64,
	sellPrice float64,
	orderID string,
) {
	od.mu.Lock()
	defer od.mu.Unlock()

	od.recentOrders[symbol] = &RecentOrder{
		OrderID:     orderID,
		Symbol:      symbol,
		Direction:   direction,
		Quantity:    quantity,
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		PlacedAt:    time.Now().UnixMilli(),
		Status:      "PENDING",
		LastCheckAt: time.Now().UnixMilli(),
	}
}

// UpdateOrderStatus 更新订单状态
func (od *OrderDeduplicator) UpdateOrderStatus(
	orderID string,
	buyOrderID string,
	sellOrderID string,
	status string,
	failReason string,
) error {
	od.mu.Lock()
	defer od.mu.Unlock()

	// 查找订单
	for _, order := range od.recentOrders {
		if order.OrderID == orderID {
			order.Status = status
			if buyOrderID != "" {
				order.BuyOrderID = buyOrderID
			}
			if sellOrderID != "" {
				order.SellOrderID = sellOrderID
			}
			if status == "EXECUTED" {
				order.ExecutedAt = time.Now().UnixMilli()
			}
			if failReason != "" {
				order.FailReason = failReason
			}
			order.LastCheckAt = time.Now().UnixMilli()
			return nil
		}
	}

	return fmt.Errorf("order %s not found", orderID)
}

// GetRecentOrder 获取最近的订单
func (od *OrderDeduplicator) GetRecentOrder(symbol string) (*RecentOrder, bool) {
	od.mu.RLock()
	defer od.mu.RUnlock()

	order, exists := od.recentOrders[symbol]
	return order, exists
}

// NeedStatusCheck 检查订单是否需要状态验证
func (od *OrderDeduplicator) NeedStatusCheck(orderID string) bool {
	od.mu.RLock()
	defer od.mu.RUnlock()

	for _, order := range od.recentOrders {
		if order.OrderID == orderID {
			timeSinceLastCheck := time.Since(time.UnixMilli(order.LastCheckAt))
			return timeSinceLastCheck > od.OrderStateCheckTime
		}
	}

	return false
}

// CleanupExpiredOrders 清理过期订单（已成交或失败超过 1 分钟）
func (od *OrderDeduplicator) CleanupExpiredOrders() {
	od.mu.Lock()
	defer od.mu.Unlock()

	now := time.Now()
	toDelete := []string{}

	for symbol, order := range od.recentOrders {
		// 已成交或失败的订单，如果超过 1 分钟则删除
		if order.Status == "EXECUTED" || order.Status == "FAILED" {
			timeSince := now.Sub(time.UnixMilli(order.ExecutedAt))
			if timeSince > time.Minute {
				toDelete = append(toDelete, symbol)
			}
		}
	}

	for _, symbol := range toDelete {
		delete(od.recentOrders, symbol)
	}
}

// countPendingOrders 计算待成交订单数
func (od *OrderDeduplicator) countPendingOrders(symbol string) int {
	count := 0
	for _, order := range od.recentOrders {
		if order.Symbol == symbol &&
			(order.Status == "PENDING" || order.Status == "EXECUTING") {
			count++
		}
	}
	return count
}

// GetOrderHistory 获取订单历史
func (od *OrderDeduplicator) GetOrderHistory(symbol string, limit int) []*RecentOrder {
	od.mu.RLock()
	defer od.mu.RUnlock()

	var history []*RecentOrder

	// 返回所有记录中与 symbol 匹配的
	for _, order := range od.recentOrders {
		if order.Symbol == symbol {
			history = append(history, order)
		}
	}

	// 限制数量
	if len(history) > limit {
		history = history[:limit]
	}

	return history
}

// GetStats 获取统计信息
func (od *OrderDeduplicator) GetStats() *DeduplicationStats {
	od.mu.RLock()
	defer od.mu.RUnlock()

	stats := &DeduplicationStats{
		TotalOrders:   len(od.recentOrders),
		PendingCount:  0,
		ExecutedCount: 0,
		FailedCount:   0,
		BySymbol:      make(map[string]int),
	}

	for symbol, order := range od.recentOrders {
		stats.BySymbol[symbol]++

		switch order.Status {
		case "PENDING", "EXECUTING":
			stats.PendingCount++
		case "EXECUTED":
			stats.ExecutedCount++
		case "FAILED":
			stats.FailedCount++
		}
	}

	return stats
}

// DeduplicationStats 去重统计信息
type DeduplicationStats struct {
	TotalOrders   int
	PendingCount  int
	ExecutedCount int
	FailedCount   int
	BySymbol      map[string]int // symbol -> count
}

// ===== OrderValidator 订单验证器 =====

// OrderValidator 订单验证器 - 验证订单成交并处理异常
type OrderValidator struct {
	mu sync.RWMutex

	binanceClient OrderClient
	bybitClient   OrderClient
	deduplicator  *OrderDeduplicator

	// 配置
	MaxRetries        int
	RetryInterval     time.Duration
	ValidationTimeout time.Duration
}

// NewOrderValidator 创建订单验证器
func NewOrderValidator(
	binanceClient OrderClient,
	bybitClient OrderClient,
	deduplicator *OrderDeduplicator,
) *OrderValidator {
	return &OrderValidator{
		binanceClient:     binanceClient,
		bybitClient:       bybitClient,
		deduplicator:      deduplicator,
		MaxRetries:        5,
		RetryInterval:     3 * time.Second,
		ValidationTimeout: 2 * time.Minute,
	}
}

// ValidateOrder 验证订单状态
func (ov *OrderValidator) ValidateOrder(
	ctx context.Context,
	orderID string,
	symbol string,
	expectedSide string,
) (*OrderValidationResult, error) {
	recent, exists := ov.deduplicator.GetRecentOrder(symbol)
	if !exists {
		return nil, fmt.Errorf("order record not found for %s", symbol)
	}

	result := &OrderValidationResult{
		OrderID:        orderID,
		Symbol:         symbol,
		StartTime:      time.UnixMilli(recent.PlacedAt),
		ValidationTime: time.Now(),
	}

	// 查询 Binance 订单状态
	if recent.BuyOrderID != "" {
		status, err := ov.binanceClient.GetOrderStatus(ctx, symbol, recent.BuyOrderID)
		if err == nil {
			result.BinanceStatus = status.Status
			result.BinanceExecutedQty = status.ExecutedQuantity
			result.BinanceAvgPrice = status.AvgExecutedPrice
		} else {
			result.BinanceStatus = "UNKNOWN"
		}
	}

	// 查询 Bybit 订单状态
	if recent.SellOrderID != "" {
		status, err := ov.bybitClient.GetOrderStatus(ctx, symbol, recent.SellOrderID)
		if err == nil {
			result.BybitStatus = status.Status
			result.BybitExecutedQty = status.ExecutedQuantity
			result.BybitAvgPrice = status.AvgExecutedPrice
		} else {
			result.BybitStatus = "UNKNOWN"
		}
	}

	// 判断订单是否完全成交
	result.IsFullyExecuted = result.BinanceStatus == "FILLED" && result.BybitStatus == "FILLED"

	// 判断是否有一侧失败
	result.HasFailure = (result.BinanceStatus == "CANCELED" || result.BybitStatus == "CANCELED")

	// 更新去重器中的状态
	var status string
	if result.IsFullyExecuted {
		status = "EXECUTED"
	} else if result.HasFailure {
		status = "FAILED"
	} else {
		status = "EXECUTING"
	}

	_ = ov.deduplicator.UpdateOrderStatus(
		orderID,
		recent.BuyOrderID,
		recent.SellOrderID,
		status,
		"",
	)

	return result, nil
}

// OrderValidationResult 订单验证结果
type OrderValidationResult struct {
	OrderID        string
	Symbol         string
	StartTime      time.Time
	ValidationTime time.Time

	// Binance 订单状态
	BinanceStatus      string
	BinanceExecutedQty float64
	BinanceAvgPrice    float64

	// Bybit 订单状态
	BybitStatus      string
	BybitExecutedQty float64
	BybitAvgPrice    float64

	// 结果判断
	IsFullyExecuted bool
	HasFailure      bool
}

// ===== DuplicateOrderGuard 完整守卫 =====

// DuplicateOrderGuard 完整的重复下单防护机制
type DuplicateOrderGuard struct {
	deduplicator *OrderDeduplicator
	validator    *OrderValidator

	mu sync.RWMutex

	// 黑名单（短期内失败的交易对）
	failedSymbols map[string]time.Time
	blacklistTTL  time.Duration
}

// NewDuplicateOrderGuard 创建完整防护
func NewDuplicateOrderGuard(
	binanceClient OrderClient,
	bybitClient OrderClient,
) *DuplicateOrderGuard {
	dedup := NewOrderDeduplicator()
	validator := NewOrderValidator(binanceClient, bybitClient, dedup)

	return &DuplicateOrderGuard{
		deduplicator:  dedup,
		validator:     validator,
		failedSymbols: make(map[string]time.Time),
		blacklistTTL:  30 * time.Second,
	}
}

// CanPlaceOrder 完整的下单前检查
func (dog *DuplicateOrderGuard) CanPlaceOrder(
	symbol string,
	direction string,
	quantity float64,
) (bool, string) {
	// 检查 1: 黑名单
	dog.mu.RLock()
	if failTime, exists := dog.failedSymbols[symbol]; exists {
		if time.Since(failTime) < dog.blacklistTTL {
			dog.mu.RUnlock()
			return false, fmt.Sprintf("symbol in blacklist: failed %.1fs ago",
				time.Since(failTime).Seconds())
		}
		delete(dog.failedSymbols, symbol)
	}
	dog.mu.RUnlock()

	// 检查 2: 去重检查
	canPlace, reason := dog.deduplicator.CanPlaceOrder(symbol, direction, quantity)
	return canPlace, reason
}

// RegisterOrder 注册订单
func (dog *DuplicateOrderGuard) RegisterOrder(
	symbol string,
	direction string,
	quantity float64,
	buyPrice float64,
	sellPrice float64,
	orderID string,
) {
	dog.deduplicator.RegisterOrder(symbol, direction, quantity, buyPrice, sellPrice, orderID)
}

// MarkOrderFailure 标记订单失败
func (dog *DuplicateOrderGuard) MarkOrderFailure(
	orderID string,
	symbol string,
	failReason string,
) {
	// 更新订单状态
	_ = dog.deduplicator.UpdateOrderStatus(orderID, "", "", "FAILED", failReason)

	// 加入黑名单
	dog.mu.Lock()
	dog.failedSymbols[symbol] = time.Now()
	dog.mu.Unlock()
}

// MarkOrderSuccess 标记订单成功
func (dog *DuplicateOrderGuard) MarkOrderSuccess(
	orderID string,
	buyOrderID string,
	sellOrderID string,
) {
	_ = dog.deduplicator.UpdateOrderStatus(
		orderID,
		buyOrderID,
		sellOrderID,
		"EXECUTED",
		"",
	)
}

// ValidateAndCleanup 验证订单并清理过期数据
func (dog *DuplicateOrderGuard) ValidateAndCleanup(ctx context.Context) {
	// 清理过期订单记录
	dog.deduplicator.CleanupExpiredOrders()

	// 清理过期的黑名单
	dog.mu.Lock()
	toDelete := []string{}
	for symbol, failTime := range dog.failedSymbols {
		if time.Since(failTime) > dog.blacklistTTL {
			toDelete = append(toDelete, symbol)
		}
	}
	for _, symbol := range toDelete {
		delete(dog.failedSymbols, symbol)
	}
	dog.mu.Unlock()
}

// GetGuardStats 获取防护统计
func (dog *DuplicateOrderGuard) GetGuardStats() map[string]interface{} {
	dedupStats := dog.deduplicator.GetStats()

	dog.mu.RLock()
	blacklistCount := len(dog.failedSymbols)
	dog.mu.RUnlock()

	return map[string]interface{}{
		"total_orders":        dedupStats.TotalOrders,
		"pending_orders":      dedupStats.PendingCount,
		"executed_orders":     dedupStats.ExecutedCount,
		"failed_orders":       dedupStats.FailedCount,
		"blacklisted_symbols": blacklistCount,
		"by_symbol":           dedupStats.BySymbol,
	}
}
