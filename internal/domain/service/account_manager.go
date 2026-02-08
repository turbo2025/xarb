package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AccountInfo 账户信息
type AccountInfo struct {
	Exchange    string                    // 交易所名称 (binance, bybit)
	TotalMargin float64                   // 总保证金
	AvailMargin float64                   // 可用保证金
	UsedMargin  float64                   // 已用保证金
	Positions   map[string]*PositionInfo  // 持仓 symbol -> position
	OpenOrders  map[string]*OpenOrderInfo // 挂单 orderID -> order
	UpdatedAt   time.Time                 // 更新时间
}

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol     string    // 交易对 (e.g., BTCUSDT)
	Side       string    // 持仓方向 (LONG/SHORT)
	Quantity   float64   // 持仓量
	EntryPrice float64   // 开仓价格
	MarkPrice  float64   // 标记价格
	PnL        float64   // 未实现盈亏
	PnLRatio   float64   // 未实现盈亏率
	Leverage   float64   // 杠杆倍数
	UpdatedAt  time.Time // 更新时间
}

// OpenOrderInfo 挂单信息
type OpenOrderInfo struct {
	OrderID          string    // 订单ID
	Symbol           string    // 交易对
	Side             string    // 方向 (BUY/SELL)
	Quantity         float64   // 委托量
	Price            float64   // 委托价
	ExecutedQuantity float64   // 成交量
	Status           string    // 状态
	CreatedAt        time.Time // 创建时间
	UpdatedAt        time.Time // 更新时间
}

// OrderLog 订单日志
type OrderLog struct {
	OrderID          string    // 订单ID
	Symbol           string    // 交易对
	Side             string    // 方向
	Quantity         float64   // 数量
	Price            float64   // 价格
	AvgExecutedPrice float64   // 平均成交价
	ExecutedQty      float64   // 成交量
	Status           string    // 状态
	Fee              float64   // 手续费
	Profit           float64   // 盈亏
	CreatedAt        time.Time // 创建时间
	ClosedAt         time.Time // 关闭时间
}

// AccountClient 账户查询接口
type AccountClient interface {
	// GetAccount 获取账户信息
	GetAccount(ctx context.Context) (*AccountInfo, error)

	// GetPositions 获取持仓
	GetPositions(ctx context.Context) ([]*PositionInfo, error)

	// GetOpenOrders 获取挂单
	GetOpenOrders(ctx context.Context, symbol string) ([]*OpenOrderInfo, error)

	// GetOrderHistory 获取订单历史
	GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*OrderLog, error)

	// GetBalance 获取余额
	GetBalance(ctx context.Context) (float64, error)
}

// AccountManager 账户管理器
type AccountManager struct {
	clients map[string]AccountClient // 交易所 -> 客户端

	// 缓存
	cache     map[string]*AccountInfo // exchange -> account info
	cacheLock sync.RWMutex
	cacheTime map[string]time.Time // exchange -> last update time
	cacheTTL  time.Duration        // 缓存有效期，默认 5s

	// 订单日志
	orderLogs     map[string][]*OrderLog // exchange -> logs
	orderLogsLock sync.RWMutex
}

// NewAccountManager 创建账户管理器
func NewAccountManager(cacheTTL time.Duration) *AccountManager {
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Second
	}
	return &AccountManager{
		clients:       make(map[string]AccountClient),
		cache:         make(map[string]*AccountInfo),
		cacheLock:     sync.RWMutex{},
		cacheTime:     make(map[string]time.Time),
		cacheTTL:      cacheTTL,
		orderLogs:     make(map[string][]*OrderLog),
		orderLogsLock: sync.RWMutex{},
	}
}

// RegisterClient 注册交易所客户端
func (m *AccountManager) RegisterClient(exchange string, client AccountClient) error {
	if exchange == "" || client == nil {
		return fmt.Errorf("invalid exchange or client")
	}
	m.clients[exchange] = client
	return nil
}

// GetAccount 获取账户信息（带缓存）
func (m *AccountManager) GetAccount(ctx context.Context, exchange string) (*AccountInfo, error) {
	// 检查缓存
	m.cacheLock.RLock()
	cached, exists := m.cache[exchange]
	lastUpdate := m.cacheTime[exchange]
	m.cacheLock.RUnlock()

	if exists && time.Since(lastUpdate) < m.cacheTTL {
		return cached, nil
	}

	// 获取客户端
	client, ok := m.clients[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange %s not registered", exchange)
	}

	// 查询API
	account, err := client.GetAccount(ctx)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	m.cacheLock.Lock()
	m.cache[exchange] = account
	m.cacheTime[exchange] = time.Now()
	m.cacheLock.Unlock()

	return account, nil
}

// GetPositions 获取持仓（带缓存）
func (m *AccountManager) GetPositions(ctx context.Context, exchange string) ([]*PositionInfo, error) {
	account, err := m.GetAccount(ctx, exchange)
	if err != nil {
		return nil, err
	}

	positions := make([]*PositionInfo, 0, len(account.Positions))
	for _, pos := range account.Positions {
		positions = append(positions, pos)
	}
	return positions, nil
}

// GetOpenOrders 获取挂单
func (m *AccountManager) GetOpenOrders(ctx context.Context, exchange string, symbol string) ([]*OpenOrderInfo, error) {
	client, ok := m.clients[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange %s not registered", exchange)
	}

	orders, err := client.GetOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetOrderHistory 获取订单历史
func (m *AccountManager) GetOrderHistory(ctx context.Context, exchange string, symbol string, limit int) ([]*OrderLog, error) {
	client, ok := m.clients[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange %s not registered", exchange)
	}

	logs, err := client.GetOrderHistory(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}

	// 保存到本地日志
	m.orderLogsLock.Lock()
	m.orderLogs[exchange] = logs
	m.orderLogsLock.Unlock()

	return logs, nil
}

// GetBalance 获取余额
func (m *AccountManager) GetBalance(ctx context.Context, exchange string) (float64, error) {
	client, ok := m.clients[exchange]
	if !ok {
		return 0, fmt.Errorf("exchange %s not registered", exchange)
	}

	return client.GetBalance(ctx)
}

// GetAllMargin 获取所有交易所的总保证金
func (m *AccountManager) GetAllMargin(ctx context.Context) (map[string]*AccountInfo, error) {
	results := make(map[string]*AccountInfo)
	errors := make(map[string]error)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发查询所有交易所
	for exchange := range m.clients {
		wg.Add(1)
		go func(ex string) {
			defer wg.Done()
			account, err := m.GetAccount(ctx, ex)
			mu.Lock()
			if err != nil {
				errors[ex] = err
			} else {
				results[ex] = account
			}
			mu.Unlock()
		}(exchange)
	}
	wg.Wait()

	// 如果有错误则返回
	if len(errors) > 0 {
		return results, fmt.Errorf("failed to get account info: %v", errors)
	}

	return results, nil
}

// ClearCache 清空缓存
func (m *AccountManager) ClearCache() {
	m.cacheLock.Lock()
	m.cache = make(map[string]*AccountInfo)
	m.cacheTime = make(map[string]time.Time)
	m.cacheLock.Unlock()
}

// InvalidateCache 使缓存失效
func (m *AccountManager) InvalidateCache(exchange string) {
	m.cacheLock.Lock()
	delete(m.cache, exchange)
	delete(m.cacheTime, exchange)
	m.cacheLock.Unlock()
}

// GetOrderLogs 获取保存的订单日志
func (m *AccountManager) GetOrderLogs(exchange string) []*OrderLog {
	m.orderLogsLock.RLock()
	defer m.orderLogsLock.RUnlock()

	logs, ok := m.orderLogs[exchange]
	if !ok {
		return []*OrderLog{}
	}

	result := make([]*OrderLog, len(logs))
	copy(result, logs)
	return result
}

// CalculateTotalProfit 计算总盈亏
func (m *AccountManager) CalculateTotalProfit(exchange string) float64 {
	logs := m.GetOrderLogs(exchange)

	totalProfit := 0.0
	for _, log := range logs {
		totalProfit += log.Profit
	}

	return totalProfit
}

// AccountRiskMetrics 账户风险指标
type AccountRiskMetrics struct {
	TotalMargin   float64 // 总保证金
	UsedMargin    float64 // 已用保证金
	AvailMargin   float64 // 可用保证金
	MarginRatio   float64 // 保证金率 (used/total)
	TotalPnL      float64 // 总盈亏
	TotalPnLRatio float64 // 总盈亏率
	RiskLevel     string  // 风险等级 (low/medium/high/critical)
}

// GetAccountRiskMetrics 获取账户风险指标
func (m *AccountManager) GetRiskMetrics(ctx context.Context, exchange string) (*AccountRiskMetrics, error) {
	account, err := m.GetAccount(ctx, exchange)
	if err != nil {
		return nil, err
	}

	metrics := &AccountRiskMetrics{
		TotalMargin: account.TotalMargin,
		UsedMargin:  account.UsedMargin,
		AvailMargin: account.AvailMargin,
	}

	if account.TotalMargin > 0 {
		metrics.MarginRatio = account.UsedMargin / account.TotalMargin
	}

	// 计算总盈亏
	totalPnL := 0.0
	for _, pos := range account.Positions {
		totalPnL += pos.PnL
	}
	metrics.TotalPnL = totalPnL

	if account.TotalMargin > 0 {
		metrics.TotalPnLRatio = totalPnL / account.TotalMargin
	}

	// 判断风险等级
	if metrics.MarginRatio > 0.9 {
		metrics.RiskLevel = "critical"
	} else if metrics.MarginRatio > 0.75 {
		metrics.RiskLevel = "high"
	} else if metrics.MarginRatio > 0.5 {
		metrics.RiskLevel = "medium"
	} else {
		metrics.RiskLevel = "low"
	}

	return metrics, nil
}
