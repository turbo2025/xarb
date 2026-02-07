package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OrderManager 订单管理器 - 执行和跟踪套利订单
type OrderManager struct {
	mu sync.RWMutex

	// REST 客户端（用于下单）
	binanceClient OrderClient            // Binance 下单客户端 (已弃用，使用 clients 代替)
	bybitClient   OrderClient            // Bybit 下单客户端 (已弃用，使用 clients 代替)
	clients       map[string]OrderClient // 所有交易所的下单客户端 exchange -> OrderClient

	// 风险管理
	marginMgr *MarginManager       // 保证金管理器
	dupGuard  *DuplicateOrderGuard // 防重复下单

	// 订单记录
	orders    map[string]*OrderDetails // OrderID -> OrderDetails
	positions map[string]*Position     // Symbol -> Position

	// 配置
	DefaultQuantity float64 // 默认交易数量
	MaxRetries      int
	RetryDelay      time.Duration
	Leverage        float64 // 杠杆倍数（默认 1，表示现货）
}

// OrderClient 订单客户端接口
type OrderClient interface {
	// PlaceOrder 下单
	// side: "BUY" 或 "SELL"
	// quantity: 交易数量
	// price: 价格（市价单为 0）
	PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error)

	// CancelOrder 撤销订单
	CancelOrder(ctx context.Context, symbol string, orderId string) error

	// GetOrderStatus 查询订单状态
	GetOrderStatus(ctx context.Context, symbol string, orderId string) (*OrderStatus, error)

	// GetFundingRate 获取融资费率
	GetFundingRate(ctx context.Context, symbol string) (float64, error)
}

// OrderStatus 订单状态
type OrderStatus struct {
	OrderID          string
	Symbol           string
	Side             string // "BUY" or "SELL"
	Quantity         float64
	ExecutedQuantity float64
	Price            float64
	AvgExecutedPrice float64
	Status           string // "PENDING", "FILLED", "PARTIALLY_FILLED", "CANCELED"
	CreatedAt        int64
	UpdatedAt        int64
}

// Position 仓位信息
type Position struct {
	Symbol        string
	Quantity      float64
	EntryPrice    float64
	Direction     string // "LONG" or "SHORT"
	UnrealizedPnL float64
	ExchangePair  string // "BINANCE" or "BYBIT"
	OpenedAt      int64
	Notes         string
}

// NewOrderManager 创建订单管理器
func NewOrderManager(binanceClient, bybitClient OrderClient) *OrderManager {
	clients := make(map[string]OrderClient)

	// 将传入的客户端添加到统一管理的 clients map
	if binanceClient != nil {
		clients["BINANCE"] = binanceClient
	}
	if bybitClient != nil {
		clients["BYBIT"] = bybitClient
	}

	return &OrderManager{
		binanceClient:   binanceClient,
		bybitClient:     bybitClient,
		clients:         clients,                                            // 新增：统一的交易所客户端映射
		marginMgr:       NewMarginManager(10000),                            // 默认 10000 USD 保证金
		dupGuard:        NewDuplicateOrderGuard(binanceClient, bybitClient), // 防重复
		orders:          make(map[string]*OrderDetails),
		positions:       make(map[string]*Position),
		DefaultQuantity: 1.0,
		MaxRetries:      3,
		RetryDelay:      time.Second * 2,
		Leverage:        1.0, // 现货模式，无杠杆
	}
}

// NewOrderManagerWithClients 新构造函数：创建 OrderManager 并支持任意数量的交易所
// clients: exchange name -> OrderClient 映射（例如 "BINANCE", "BYBIT", "OKX" 等）
func NewOrderManagerWithClients(clients map[string]OrderClient) *OrderManager {
	if len(clients) == 0 {
		return nil
	}

	// 为了兼容性，尝试从 clients 中提取 binance 和 bybit
	var binanceClient, bybitClient OrderClient
	if bc, ok := clients["BINANCE"]; ok {
		binanceClient = bc
	}
	if bc, ok := clients["BYBIT"]; ok {
		bybitClient = bc
	}

	return &OrderManager{
		binanceClient:   binanceClient,
		bybitClient:     bybitClient,
		clients:         clients,
		marginMgr:       NewMarginManager(10000),
		dupGuard:        NewDuplicateOrderGuard(binanceClient, bybitClient), // 保留 binance/bybit 用于防重复
		orders:          make(map[string]*OrderDetails),
		positions:       make(map[string]*Position),
		DefaultQuantity: 1.0,
		MaxRetries:      3,
		RetryDelay:      time.Second * 2,
		Leverage:        1.0,
	}
}

// SetMarginManager 设置保证金管理器
func (om *OrderManager) SetMarginManager(marginMgr *MarginManager) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.marginMgr = marginMgr
}

// ExecuteArbitrage 执行套利交易（含保证金检查）
func (om *OrderManager) ExecuteArbitrage(
	ctx context.Context,
	executor *ArbitrageExecutor,
	symbol string,
	binancePrice float64,
	bybitPrice float64,
	quantity float64,
) (*ArbitrageExecution, error) {
	om.mu.Lock()

	// 1. 防重复下单检查（最早检查，避免锁定资源）
	canPlace, reason := om.dupGuard.CanPlaceOrder(symbol, "ARBITRAGE", quantity)
	if !canPlace {
		om.mu.Unlock()
		return nil, fmt.Errorf("duplicate order prevention: %s", reason)
	}

	// 2. 分析机会
	orderDetails, err := executor.CalculateOrderDetails(symbol, binancePrice, bybitPrice, quantity)
	if err != nil {
		om.mu.Unlock()
		return nil, fmt.Errorf("opportunity analysis failed: %w", err)
	}

	// 3. 计算所需保证金
	requiredMargin := CalculateRequiredMargin(binancePrice, quantity, om.Leverage)

	// 4. 检查保证金（关键：提前检查）
	if err := om.marginMgr.CanExecuteOrder(symbol, requiredMargin); err != nil {
		om.mu.Unlock()
		return nil, fmt.Errorf("margin check failed: %w", err)
	}

	// 4. 注册订单到保证金管理器和防重复
	executionID := fmt.Sprintf("%s_%d", symbol, time.Now().UnixNano())
	if err := om.marginMgr.RegisterOrder(
		executionID,
		symbol,
		orderDetails.Direction,
		quantity,
		requiredMargin,
		orderDetails.ExpectedProfit,
	); err != nil {
		om.mu.Unlock()
		return nil, fmt.Errorf("failed to register order: %w", err)
	}

	// 在防重复系统中注册（使用相同的 executionID）
	om.dupGuard.RegisterOrder(symbol, orderDetails.Direction, quantity, binancePrice, bybitPrice, executionID)

	// 5. 判断交易方向
	var (
		buyEx      OrderClient
		sellEx     OrderClient
		buySymbol  string
		sellSymbol string
	)

	if orderDetails.Direction == "BUY_BINANCE_SELL_BYBIT" {
		buyEx = om.binanceClient
		sellEx = om.bybitClient
		buySymbol = symbol
		sellSymbol = symbol
	} else {
		buyEx = om.bybitClient
		sellEx = om.binanceClient
		buySymbol = symbol
		sellSymbol = symbol
	}
	om.mu.Unlock()

	// 6. 下单（市价单以快速成交）
	buyOrderID, err := om.retryPlaceOrder(ctx, buyEx, buySymbol, "BUY", quantity, orderDetails.BuyPrice, true)
	if err != nil {
		// 回滚：取消订单注册并标记失败
		om.marginMgr.mu.Lock()
		delete(om.marginMgr.ActiveOrders, executionID)
		om.marginMgr.mu.Unlock()
		// 标记重复防护为失败
		om.dupGuard.MarkOrderFailure(executionID, symbol, fmt.Sprintf("buy order failed: %v", err))
		return nil, fmt.Errorf("buy order failed: %w", err)
	}

	// 7. 验证买单成交
	buyStatus, err := om.retryGetOrderStatus(ctx, buyEx, buySymbol, buyOrderID)
	if err != nil || buyStatus.ExecutedQuantity == 0 {
		// 标记重复防护为失败（买单未成交）
		om.dupGuard.MarkOrderFailure(executionID, symbol, fmt.Sprintf("buy order not executed: %v", err))
		return nil, fmt.Errorf("buy order not executed: %w", err)
	}

	// 8. 立即卖出（注意：实际成交价可能不同）
	sellOrderID, err := om.retryPlaceOrder(ctx, sellEx, sellSymbol, "SELL", buyStatus.ExecutedQuantity, orderDetails.SellPrice, true)
	if err != nil {
		// 标记重复防护为失败（卖单失败）
		om.dupGuard.MarkOrderFailure(executionID, symbol, fmt.Sprintf("sell order failed: %v", err))
		return nil, fmt.Errorf("sell order failed: %w", err)
	}

	// 9. 更新持仓到保证金管理器
	om.marginMgr.ExecuteOrder(
		executionID,
		buyOrderID,
		sellOrderID,
		symbol,
		buyStatus.ExecutedQuantity,
		buyStatus.AvgExecutedPrice,
		orderDetails.SellPrice,
	)

	// 10. 标记订单成功
	om.dupGuard.MarkOrderSuccess(executionID, buyOrderID, sellOrderID)

	// 11. 记录执行结果
	execution := &ArbitrageExecution{
		Symbol:             symbol,
		Direction:          orderDetails.Direction,
		Quantity:           quantity,
		BuyOrderID:         buyOrderID,
		SellOrderID:        sellOrderID,
		ExpectedProfit:     orderDetails.ExpectedProfit,
		ExpectedProfitRate: orderDetails.NetProfitRate,
		ExecutedAt:         time.Now().UnixMilli(),
	}

	return execution, nil
}

// ExecuteArbitrage_old 执行套利交易
func (om *OrderManager) ExecuteArbitrage_old(
	ctx context.Context,
	executor *ArbitrageExecutor,
	symbol string,
	binancePrice float64,
	bybitPrice float64,
	quantity float64,
) (*ArbitrageExecution, error) {
	om.mu.Lock()
	defer om.mu.Unlock()

	// 1. 分析机会
	orderDetails, err := executor.CalculateOrderDetails(symbol, binancePrice, bybitPrice, quantity)
	if err != nil {
		return nil, fmt.Errorf("opportunity analysis failed: %w", err)
	}

	// 2. 判断交易方向
	var (
		buyEx      OrderClient
		sellEx     OrderClient
		buySymbol  string
		sellSymbol string
	)

	if orderDetails.Direction == "BUY_BINANCE_SELL_BYBIT" {
		buyEx = om.binanceClient
		sellEx = om.bybitClient
		buySymbol = symbol
		sellSymbol = symbol
	} else {
		buyEx = om.bybitClient
		sellEx = om.binanceClient
		buySymbol = symbol
		sellSymbol = symbol
	}

	// 3. 下单（市价单以快速成交）
	buyOrderID, err := om.retryPlaceOrder(ctx, buyEx, buySymbol, "BUY", quantity, orderDetails.BuyPrice, true)
	if err != nil {
		return nil, fmt.Errorf("buy order failed: %w", err)
	}

	// 4. 验证买单成交
	buyStatus, err := om.retryGetOrderStatus(ctx, buyEx, buySymbol, buyOrderID)
	if err != nil || buyStatus.ExecutedQuantity == 0 {
		// 回滚：取消卖单
		return nil, fmt.Errorf("buy order not executed: %w", err)
	}

	// 5. 立即卖出（注意：实际成交价可能不同）
	sellOrderID, err := om.retryPlaceOrder(ctx, sellEx, sellSymbol, "SELL", buyStatus.ExecutedQuantity, orderDetails.SellPrice, true)
	if err != nil {
		// 风险：已经买入但卖出失败，需要处理
		return nil, fmt.Errorf("sell order failed: %w", err)
	}

	// 6. 记录执行结果
	execution := &ArbitrageExecution{
		Symbol:             symbol,
		Direction:          orderDetails.Direction,
		Quantity:           quantity,
		BuyOrderID:         buyOrderID,
		SellOrderID:        sellOrderID,
		ExpectedProfit:     orderDetails.ExpectedProfit,
		ExpectedProfitRate: orderDetails.NetProfitRate,
		ExecutedAt:         time.Now().UnixMilli(),
	}

	return execution, nil
}

// retryPlaceOrder 重试下单
func (om *OrderManager) retryPlaceOrder(
	ctx context.Context,
	client OrderClient,
	symbol string,
	side string,
	quantity float64,
	price float64,
	isMarket bool,
) (string, error) {
	var lastErr error
	for i := 0; i < om.MaxRetries; i++ {
		orderID, err := client.PlaceOrder(ctx, symbol, side, quantity, price, isMarket)
		if err == nil {
			return orderID, nil
		}
		lastErr = err
		time.Sleep(om.RetryDelay)
	}
	return "", lastErr
}

// retryGetOrderStatus 重试查询订单状态
func (om *OrderManager) retryGetOrderStatus(
	ctx context.Context,
	client OrderClient,
	symbol string,
	orderID string,
) (*OrderStatus, error) {
	var lastErr error
	for i := 0; i < om.MaxRetries; i++ {
		status, err := client.GetOrderStatus(ctx, symbol, orderID)
		if err == nil {
			return status, nil
		}
		lastErr = err
		time.Sleep(om.RetryDelay)
	}
	return nil, lastErr
}

// ArbitrageExecution 套利执行记录
type ArbitrageExecution struct {
	Symbol             string
	Direction          string // "BUY_BINANCE_SELL_BYBIT" or "BUY_BYBIT_SELL_BINANCE"
	Quantity           float64
	BuyOrderID         string
	SellOrderID        string
	ExpectedProfit     float64
	ExpectedProfitRate float64
	ActualProfit       float64 // 实际利润（成交后计算）
	ExecutedAt         int64   // Unix milliseconds
	ClosedAt           int64   // 关闭时间
	Status             string  // "PENDING", "EXECUTED", "FAILED", "PARTIALLY_CLOSED"
}

// GetPosition 获取仓位
func (om *OrderManager) GetPosition(symbol string) (*Position, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	pos, exists := om.positions[symbol]
	return pos, exists
}

// UpdatePosition 更新仓位
func (om *OrderManager) UpdatePosition(symbol string, position *Position) {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.positions[symbol] = position
}

// ListPositions 列出所有仓位
func (om *OrderManager) ListPositions() []*Position {
	om.mu.RLock()
	defer om.mu.RUnlock()

	positions := make([]*Position, 0, len(om.positions))
	for _, pos := range om.positions {
		positions = append(positions, pos)
	}
	return positions
}

// ClosePosition 平仓
func (om *OrderManager) ClosePosition(ctx context.Context, symbol string, currentPrice float64) error {
	om.mu.Lock()
	position, exists := om.positions[symbol]
	om.mu.Unlock()

	if !exists {
		return fmt.Errorf("position not found for symbol %s", symbol)
	}

	// 根据方向选择客户端
	var client OrderClient
	if position.ExchangePair == "BINANCE" {
		client = om.binanceClient
	} else {
		client = om.bybitClient
	}

	// 平仓（反向操作）
	side := "SELL"
	if position.Direction == "SHORT" {
		side = "BUY"
	}

	orderID, err := om.retryPlaceOrder(ctx, client, symbol, side, position.Quantity, currentPrice, true)
	if err != nil {
		return fmt.Errorf("close position order failed: %w", err)
	}

	// 更新仓位状态
	om.mu.Lock()
	delete(om.positions, symbol)
	om.mu.Unlock()

	fmt.Printf("Position closed for %s (Order ID: %s)\n", symbol, orderID)
	return nil
}

// MonitorAndClosePositions 监控持仓，自动平仓
// 定期调用此函数来检查是否需要挂单平仓或市价平仓
func (om *OrderManager) MonitorAndClosePositions(
	ctx context.Context,
	priceFeeds map[string][2]float64, // symbol -> [buyPrice, sellPrice]
) error {
	// 定期验证和清理重复订单列表
	om.dupGuard.ValidateAndCleanup(ctx)

	for symbol, prices := range priceFeeds {
		buyPrice := prices[0]
		sellPrice := prices[1]

		// 更新持仓利润
		pos, err := om.marginMgr.UpdatePositionProfit(symbol, buyPrice, sellPrice)
		if err != nil {
			continue // 没有该持仓
		}

		if pos.Status != "OPEN" {
			continue // 持仓已关闭
		}

		// 检查是否需要止损
		needStop, _, _ := om.marginMgr.NeedStopLoss(symbol)
		if needStop {
			// 利润下跌到 5% 以下，需要市价平仓
			fmt.Printf("⚠️  Symbol %s profit dropped to %.2f%%, triggering market close\n",
				symbol, pos.CurrentProfitRate)

			// 市价平仓
			if err := om.marginMgr.ClosePositionWithMarket(symbol, sellPrice); err != nil {
				fmt.Printf("Failed to market close %s: %v\n", symbol, err)
			}

			// 实际执行平仓
			sellClient := om.binanceClient // 假设卖出端是 Binance
			closeOrderID, _ := om.retryPlaceOrder(ctx, sellClient, symbol, "SELL", pos.Quantity, sellPrice, true)

			// 标记平仓
			realizedProfit := pos.CurrentProfit
			_ = om.marginMgr.MarkPositionClosed(symbol, realizedProfit)

			fmt.Printf("✓ Market closed %s (Order: %s), Realized PnL: %.2f USD\n",
				symbol, closeOrderID, realizedProfit)
		}
	}

	return nil
}

// TryLimitClose 尝试挂单平仓（限价单）
// 在持仓实现初始利润后，设置一个高于预期的限价卖出
func (om *OrderManager) TryLimitClose(
	ctx context.Context,
	symbol string,
	limitPrice float64,
) error {
	pos, err := om.marginMgr.GetPositionStatus(symbol)
	if err != nil {
		return err
	}

	if pos.Status != "OPEN" {
		return fmt.Errorf("position %s is not open", symbol)
	}

	// 设置挂单平仓
	if err := om.marginMgr.ClosePositionWithLimit(symbol, limitPrice); err != nil {
		return err
	}

	// 实际下单
	client := om.binanceClient // 假设卖出端
	closeOrderID, _ := om.retryPlaceOrder(ctx, client, symbol, "SELL", pos.Quantity, limitPrice, false)

	fmt.Printf("Limit close order placed for %s at %.2f (Order: %s)\n", symbol, limitPrice, closeOrderID)

	return nil
}

// GetAccountStatus 获取账户状态（保证金、风险等）
func (om *OrderManager) GetAccountStatus() *AccountStatus {
	marginStatus := om.marginMgr.GetMarginStatus()
	riskLevel := om.marginMgr.RiskLevel()

	return &AccountStatus{
		TotalMargin:       marginStatus.TotalMargin,
		UsedMargin:        marginStatus.UsedMargin,
		AvailableMargin:   marginStatus.AvailableMargin,
		MarginUsageRate:   marginStatus.UsageRate,
		RiskLevel:         riskLevel,
		ActiveOrderCount:  marginStatus.ActiveOrderCount,
		OpenPositionCount: marginStatus.OpenPositionCount,
		RealizedPnL:       marginStatus.RealizedPnL,
	}
}

// AccountStatus 账户状态
type AccountStatus struct {
	TotalMargin       float64
	UsedMargin        float64
	AvailableMargin   float64
	MarginUsageRate   float64 // 百分比
	RiskLevel         string  // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	ActiveOrderCount  int
	OpenPositionCount int
	RealizedPnL       float64
}

// GetOrderStatus 查询订单状态（通过 REST API）
func (om *OrderManager) GetOrderStatus(ctx context.Context, exchange string, symbol string, orderId string) (*OrderStatus, error) {
	var client OrderClient

	switch exchange {
	case "binance", "Binance":
		client = om.binanceClient
	case "bybit", "Bybit":
		client = om.bybitClient
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchange)
	}

	if client == nil {
		return nil, fmt.Errorf("client not initialized for %s", exchange)
	}

	// 调用 REST API 查询订单状态
	status, err := client.GetOrderStatus(ctx, symbol, orderId)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s order %s: %w", exchange, orderId, err)
	}

	return status, nil
}
