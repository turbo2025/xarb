# 防重复下单系统 (Duplicate Order Prevention)

## 概述

为了防止交易系统在高频运行或网络延迟情况下重复下单，实现了一套完整的防重复（deduplication）和订单追踪机制。该系统涵盖：

1. **时间窗口去重**：同一交易对在5秒内不允许重复下单
2. **冷却期限制**：同一交易对需要10秒冷却期才能再次下单
3. **待成交数量限制**：单个交易对最多允许2个待成交订单
4. **订单状态验证**：通过REST API验证实际订单成交状态
5. **失败黑名单**：失败订单的交易对会被列入30秒黑名单

## 架构设计

```
OrderManager (订单管理器)
    ├─ DuplicateOrderGuard (防重复防护)
    │   ├─ OrderDeduplicator (去重逻辑)
    │   ├─ OrderValidator (订单验证)
    │   └─ Blacklist (黑名单管理)
    ├─ MarginManager (保证金管理)
    └─ REST Clients (Binance, Bybit)
```

## 核心组件

### 1. DuplicateOrderGuard (完整防护框架)

主要职责：
- 整合去重、验证、黑名单三层防护
- 提供统一的接口给OrderManager调用

```go
type DuplicateOrderGuard struct {
    deduplicator *OrderDeduplicator  // 去重核心
    validator    *OrderValidator     // 订单验证
    failedSymbols map[string]time.Time // 黑名单
    blacklistTTL time.Duration        // 黑名单有效期（30秒）
}
```

**关键方法：**
- `CanPlaceOrder(symbol, direction, quantity)` - 下单前检查
- `RegisterOrder(symbol, direction, quantity, buyPrice, sellPrice, orderID)` - 注册新订单
- `MarkOrderSuccess(orderID, buyOrderID, sellOrderID)` - 标记成功
- `MarkOrderFailure(orderID, symbol, failReason)` - 标记失败
- `ValidateAndCleanup(ctx)` - 定期验证和清理

### 2. OrderDeduplicator (核心去重逻辑)

**配置参数：**
```go
DeduplicationWindow: 5秒    // 相同方向不重复
CooldownPeriod: 10秒        // 同对冷却期
MaxOrdersPerSymbol: 2       // 单对最多待成交订单
OrderStateCheckTime: 30秒   // 状态检查周期
```

**订单生命周期：**
```
PENDING -> EXECUTING -> EXECUTED / FAILED
```

### 3. OrderValidator (订单验证)

通过REST API查询订单实际成交状态：
- Binance：查询买单和卖单的已成交数量
- Bybit：查询线性永续合约订单状态
- 对比记录状态，发现不一致时发出告警

## 集成流程

### ExecuteArbitrage方法中的集成

```go
func (om *OrderManager) ExecuteArbitrage(...) (*ArbitrageExecution, error) {
    // 1. 防重复检查（最优先）
    canPlace, reason := om.dupGuard.CanPlaceOrder(symbol, "ARBITRAGE", quantity)
    if !canPlace {
        return nil, fmt.Errorf("duplicate order prevention: %s", reason)
    }

    // 2. 分析机会、检查保证金...

    // 3. 在防重复系统中注册
    om.dupGuard.RegisterOrder(symbol, orderDetails.Direction, quantity, 
                              binancePrice, bybitPrice, executionID)

    // 4. 执行买卖订单...

    // 5. 如果失败，标记失败
    if err != nil {
        om.dupGuard.MarkOrderFailure(executionID, symbol, failReason)
    }

    // 6. 如果成功，标记成功
    om.dupGuard.MarkOrderSuccess(executionID, buyOrderID, sellOrderID)
}
```

### MonitorAndClosePositions方法中的集成

```go
func (om *OrderManager) MonitorAndClosePositions(...) error {
    // 定期验证订单状态和清理过期数据
    om.dupGuard.ValidateAndCleanup(ctx)

    // 监控持仓，必要时平仓...
}
```

## 防护机制详解

### 1. 时间窗口去重 (5秒)

**场景：** 同一交易对BTC/USDT，5秒内收到两次价差信号

**检查逻辑：**
```
如果 (当前时间 - 上次下单时间) < 5秒 AND 方向相同
    -> 拒绝下单，返回: "same order within deduplication window"
```

### 2. 冷却期限制 (10秒)

**场景：** BTC/USDT刚成交，立即又有价差信号

**检查逻辑：**
```
如果 订单已成交 AND (当前时间 - 成交时间) < 10秒
    -> 拒绝下单，返回: "cooldown period not met (X.Xs remaining)"
```

### 3. 待成交数量限制 (≤2个)

**场景：** 同一对有多个待成交订单

**检查逻辑：**
```
待成交订单数 = PENDING + EXECUTING 状态的订单数
如果 待成交订单数 >= 2
    -> 拒绝下单，返回: "too many pending orders"
```

### 4. 黑名单机制 (30秒)

**场景：** BTC/USDT订单因网络错误失败

**处理流程：**
1. 订单失败时调用 `MarkOrderFailure()`
2. 交易对BTC加入黑名单，TTL=30秒
3. 在黑名单有效期内的所有CanPlaceOrder检查都会拒绝
4. 30秒后自动从黑名单移除

## 订单追踪数据结构

### RecentOrder (最近订单记录)

```go
type RecentOrder struct {
    OrderID     string      // 订单唯一ID
    Symbol      string      // 交易对（如BTC）
    Direction   string      // 交易方向（BUY_BINANCE_SELL_BYBIT等）
    Quantity    float64     // 交易数量
    BuyPrice    float64     // 买入价格
    SellPrice   float64     // 卖出价格
    PlacedAt    int64       // 下单时间（毫秒）
    BuyOrderID  string      // Binance订单ID
    SellOrderID string      // Bybit订单ID
    Status      string      // PENDING/EXECUTING/EXECUTED/FAILED
    FailReason  string      // 失败原因
    ExecutedAt  int64       // 成交时间
    LastCheckAt int64       // 上次状态检查时间
}
```

## 配置建议

### 保守配置（推荐用于生产环境）
```go
DeduplicationWindow: 5秒   // 同向禁止
CooldownPeriod: 10秒       // 必须冷却
MaxOrdersPerSymbol: 2      // 最多2个待成交
OrderStateCheckTime: 30秒  // 30秒验证一次
BlacklistTTL: 30秒         // 失败后隔离30秒
```

### 激进配置（用于高频交易）
```go
DeduplicationWindow: 2秒
CooldownPeriod: 5秒
MaxOrdersPerSymbol: 3
OrderStateCheckTime: 10秒
BlacklistTTL: 15秒
```

## 日志和监控

### 获取防护统计

```go
stats := orderManager.dupGuard.GetGuardStats()
// 返回:
// {
//   "total_orders": 150,
//   "pending_orders": 3,
//   "executed_orders": 140,
//   "failed_orders": 7,
//   "blacklisted_symbols": 1,
//   "by_symbol": {
//     "BTC": {"pending": 1, "executed": 50, "failed": 2},
//     "ETH": {"pending": 2, "executed": 90, "failed": 5}
//   }
// }
```

### 错误返回示例

```
✗ duplicate order prevention: same order within deduplication window (2.3s ago, direction: ARBITRAGE)
✗ duplicate order prevention: cooldown period not met (3.5s remaining)
✗ duplicate order prevention: too many pending orders (2/2)
✗ duplicate order prevention: symbol in blacklist: failed 15.2s ago
```

## 风险提示

1. **时间同步**：系统依赖准确的系统时间，建议定期与NTP服务器同步
2. **API延迟**：订单验证依赖REST API，高延迟时可能导致验证不及时
3. **网络故障**：黑名单机制会保守处理失败，防止连锁失败
4. **订单匹配**：确保buyOrderID和sellOrderID正确匹配，否则平仓会失败

## 测试建议

```go
// 测试1：正常下单流程
om.dupGuard.CanPlaceOrder("BTC", "ARBITRAGE", 1.0) // true

// 测试2：5秒内重复下单
time.Sleep(2 * time.Second)
om.dupGuard.CanPlaceOrder("BTC", "ARBITRAGE", 1.0) // false (dedup window)

// 测试3：冷却期限制
om.dupGuard.MarkOrderSuccess(...)
time.Sleep(5 * time.Second)
om.dupGuard.CanPlaceOrder("BTC", "ARBITRAGE", 1.0) // false (cooldown)

// 测试4：失败黑名单
om.dupGuard.MarkOrderFailure(...)
om.dupGuard.CanPlaceOrder("BTC", "ARBITRAGE", 1.0) // false (blacklist)
time.Sleep(31 * time.Second)
om.dupGuard.CanPlaceOrder("BTC", "ARBITRAGE", 1.0) // true (blacklist expired)
```

## 性能指标

- **检查开销**：<1ms (内存中查询，无IO)
- **清理周期**：30秒 (定期清理过期数据)
- **内存占用**：约2KB per symbol per order
- **并发安全**：✅ RWMutex保护，支持高并发

## 总结

防重复系统通过多层防护机制（去重窗口、冷却期、待成交限制、黑名单）和订单状态验证，确保：

✅ 消除网络延迟导致的重复下单  
✅ 防止频繁波动造成的过度交易  
✅ 及时发现订单执行异常  
✅ 从失败状态快速恢复  

该系统与 MarginManager 的风险控制、ArbitrageExecutor 的利润计算形成三层防护，共同保障交易系统的稳定性和收益性。
