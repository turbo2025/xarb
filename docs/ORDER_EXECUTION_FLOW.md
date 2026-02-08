# 完整订单执行流程 (Complete Order Execution Flow)

## 系统架构全景

```
┌─────────────────────────────────────────────────────────────────┐
│                      监控服务 (Monitor Service)                  │
│                     运行主事件循环                               │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│              WebSocket 价格订阅 (4个交易所)                      │
│  Binance | Bybit | OKX | Bitget                                 │
│  实时获取: BTC/ETH/SOL 买卖价差                                   │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│              套利分析器 (ArbitrageExecutor)                       │
│  ✓ 计算价差                                                       │
│  ✓ 扣除交易费 (Binance: 0.02%, Bybit: 0.01%)                   │
│  ✓ 扣除融资费 (Binance: 0.1%/8h, Bybit: 0.08%/8h)             │
│  ✓ 验证纯利润 >= 最小阈值 (0.1%)                               │
└────────────┬────────────────────────────────────────────────────┘
             │ 只有净利润 >= 阈值时才继续
             ▼
┌─────────────────────────────────────────────────────────────────┐
│            订单管理器 (OrderManager)                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤1: 防重复检查                                         │  │
│  │ ├─ 5秒去重窗口 (同方向禁止)                              │  │
│  │ ├─ 10秒冷却期 (下次等待)                                │  │
│  │ ├─ 最多2个待成交订单                                     │  │
│  │ └─ 30秒黑名单 (失败隔离)                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤2: 保证金检查                                         │  │
│  │ ├─ 单笔订单不超过5%保证金                                 │  │
│  │ ├─ 总待成交不超过100%保证金                               │  │
│  │ └─ 有可用资金才继续                                       │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤3: 注册订单                                           │  │
│  │ ├─ 在防重复系统中注册                                     │  │
│  │ └─ 在保证金管理中注册                                     │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤4: 执行买单                                           │  │
│  │ ├─ 在Binance下买单 (市价，快速成交)                      │  │
│  │ ├─ 轮询检查成交情况 (最多3次)                            │  │
│  │ └─ 失败时标记为失败、加入黑名单                           │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤5: 执行卖单                                           │  │
│  │ ├─ 在Bybit下卖单 (市价，快速成交)                        │  │
│  │ ├─ 轮询检查成交情况                                       │  │
│  │ └─ 失败时自动撤销前面的买单                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 步骤6: 更新持仓状态                                       │  │
│  │ ├─ 记录实际成交价格                                       │  │
│  │ ├─ 更新保证金占用                                         │  │
│  │ └─ 标记订单成功、清除黑名单                               │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│           后续监控 (Monitoring & Closing)                        │
│  ✓ 实时跟踪价差变化，更新持仓浮动利润                            │
│  ✓ 如利润下跌至峰值的95% ➜ 市价平仓 (risk stop)             │
│  ✓ 如利润达到目标 ➜ 挂单平仓 (take profit)                  │
│  ✓ 定期验证订单状态、清理过期数据                               │
└─────────────────────────────────────────────────────────────────┘
```

## 详细执行示例

### 场景1：成功的套利交易

```
时间: 10:00:00
触发: BTC价差 = 100 USDT (Binance 10000, Bybit 10100)
      净利润 = 0.8% (扣除费用后)

步骤1: CanPlaceOrder("BTC", "ARBITRAGE", 1.0)
  ✓ 5秒去重: 最后订单在9:59:55，已过期 -> PASS
  ✓ 10秒冷却: 最后成交在9:59:20，已过期 -> PASS
  ✓ 待成交数: 0 < 2 -> PASS
  ✓ 黑名单: 无 -> PASS
  ➜ 返回: (true, "")

步骤2: CanExecuteOrder("BTC", 500 USD)
  ✓ 总保证金: 10000 USD
  ✓ 已用: 2000 USD (4个其他订单)
  ✓ 可用: 8000 USD
  ✓ 需要: 500 USD (5% of 10000)
  ➜ 返回: (true, nil)

步骤3: RegisterOrder()
  ✓ 注册到防重复: OrderID=BTC_1707390000000
  ✓ 注册到保证金: ExecutionID=BTC_1707390000000
  ➜ 保证金占用: 2500 USD

步骤4: 执行买单 (Binance)
  ➜ PlaceOrder("BTC", "BUY", 1.0, 10000)
  ➜ 返回: OrderID=12345
  ➜ 确认成交: 1.0 BTC @ 10001 USDT
  ✓ 成交价稍高于报价，但合理
  ➜ 继续

步骤5: 执行卖单 (Bybit)
  ➜ PlaceOrder("BTC", "SELL", 1.0, 10100)
  ➜ 返回: OrderID=67890
  ➜ 确认成交: 1.0 BTC @ 10099 USDT
  ✓ 成交价稍低于报价，但合理
  ➜ 继续

步骤6: 更新状态
  ✓ 买入成本: 1.0 × 10001 = 10001 USDT
  ✓ 卖出收入: 1.0 × 10099 = 10099 USDT
  ✓ 毛利: 98 USDT
  ✓ 减去费用: 
    - Binance费: 10001 × 0.02% = 2.0 USDT
    - Bybit费: 10099 × 0.01% = 1.01 USDT
  ✓ 实际利润: 98 - 2.0 - 1.01 = 94.99 USDT ✓ 成功!
  ✓ 清除BTC黑名单
  ✓ 标记为EXECUTED
  ➜ 持仓记录: {Symbol: "BTC", Quantity: 1.0, ...}

步骤7: 监控持仓
  10:00:05 - 价差缩小: Binance 10001.5, Bybit 10098.5
            浮动利润更新: (10098.5 - 10001.5) × 1.0 - 费用 = 约97 USDT
  
  10:00:30 - 价差继续缩小: Binance 10010, Bybit 10090
            浮动利润下降: (10090 - 10010) × 1.0 - 费用 = 约80 USDT
            距峰值(94.99) 下跌到 84.3%
            状态: GOOD (不需要平仓)
  
  10:01:00 - 突发亏损: Binance 10050, Bybit 10040
            浮动利润: (10040 - 10050) × 1.0 - 费用 = 约-10 USDT
            距峰值下跌到 -10.5% ❌ 触发止损!
            ➜ 市价平仓: Sell 1.0 BTC @ 10040
            ➜ 本次交易结束
```

### 场景2：重复下单被阻止

```
时间: 10:00:00
第1次触发: 成功下单BTC，OrderID=BTC_1707390000000，状态=PENDING

时间: 10:00:02 (2秒后)
第2次触发: 又接收到相同的BTC价差信号

CanPlaceOrder("BTC", "ARBITRAGE", 1.0)
  ❌ 5秒去重检查失败:
  ├─ 上次下单时间: 10:00:00
  ├─ 当前时间: 10:00:02
  ├─ 时间差: 2秒 < 5秒
  ├─ 方向: 相同 (都是ARBITRAGE)
  └─ 返回: (false, "same order within deduplication window (2.0s ago)")
  ➜ 拒绝下单，记录日志: "⚠️ BTC order rejected: duplicate prevention"
```

### 场景3：失败后加入黑名单

```
时间: 10:00:00
下单: PlaceOrder("ETH", "BUY", 10.0, 2000.0)
结果: 网络超时，返回错误

MarkOrderFailure("ETH_1707390000000", "ETH", "buy order failed: context deadline exceeded")
  ✓ 状态更新: PENDING -> FAILED
  ✓ 加入黑名单: failedSymbols["ETH"] = 10:00:00 + 30秒TTL

时间: 10:00:05
新价差出现: ETH价差信号

CanPlaceOrder("ETH", "ARBITRAGE", 10.0)
  ❌ 黑名单检查失败:
  ├─ 交易对: ETH
  ├─ 加入黑名单时间: 10:00:00
  ├─ 当前时间: 10:00:05
  ├─ 已过期时长: 5秒
  ├─ 黑名单TTL: 30秒
  └─ 返回: (false, "symbol in blacklist: failed 5.0s ago")
  ➜ 继续拒绝

时间: 10:00:31
黑名单过期

CanPlaceOrder("ETH", "ARBITRAGE", 10.0)
  ✓ 黑名单已过期，自动删除
  ✓ 其他检查全部通过
  ➜ 返回: (true, "")
  ➜ 允许下单!
```

### 场景4：冷却期限制

```
时间: 10:00:00
订单成功成交:
  ✓ OrderStatus: EXECUTED
  ✓ ExecutedAt: 10:00:00

时间: 10:00:05 (5秒后)
新价差出现: BTC信号

CanPlaceOrder("BTC", "ARBITRAGE", 1.0)
  ❌ 冷却期检查失败:
  ├─ 上次成交时间: 10:00:00
  ├─ 当前时间: 10:00:05
  ├─ 已等待: 5秒
  ├─ 冷却期: 10秒
  ├─ 剩余等待: 5秒
  └─ 返回: (false, "cooldown period not met (5.0s remaining)")

时间: 10:00:11 (11秒后)
冷却期结束

CanPlaceOrder("BTC", "ARBITRAGE", 1.0)
  ✓ 上次成交: 11秒前（超过10秒冷却期）
  ✓ 其他检查全部通过
  ➜ 返回: (true, "")
  ➜ 允许下单!
```

## 核心接口

### OrderManager.ExecuteArbitrage()

```go
execution, err := orderManager.ExecuteArbitrage(
    ctx,
    executor,           // ArbitrageExecutor实例
    "BTC",              // 交易对
    10000.0,            // Binance买入价
    10100.0,            // Bybit卖出价
    1.0,                // 交易数量
)

if err != nil {
    log.Printf("❌ 执行失败: %v", err)
    // 可能的错误:
    // - "duplicate order prevention: ..."
    // - "margin check failed: ..."
    // - "opportunity analysis failed: ..."
    // - "buy order failed: ..."
    // - "sell order failed: ..."
}

// 成功时返回
log.Printf("✓ 订单执行成功")
log.Printf("  交易对: %s", execution.Symbol)
log.Printf("  方向: %s", execution.Direction)
log.Printf("  买单ID: %s", execution.BuyOrderID)
log.Printf("  卖单ID: %s", execution.SellOrderID)
log.Printf("  预期利润: $%.2f (%.2f%%)", 
           execution.ExpectedProfit, 
           execution.ExpectedProfitRate*100)
```

### OrderManager.MonitorAndClosePositions()

```go
// 定期调用（如每秒一次）
priceFeeds := map[string][2]float64{
    "BTC": {10001.0, 10099.0},  // [买价, 卖价]
    "ETH": {2001.0, 2050.0},
}

err := orderManager.MonitorAndClosePositions(ctx, priceFeeds)
if err != nil {
    log.Printf("❌ 监控失败: %v", err)
}

// 该方法会:
// 1. 验证所有订单状态
// 2. 清理过期订单和黑名单
// 3. 更新持仓浮动利润
// 4. 检查是否需要止损平仓
// 5. 检查是否可以限价平仓
```

## 监控和统计

```go
// 获取防护系统统计
stats := orderManager.dupGuard.GetGuardStats()

log.Printf("防重复系统统计:")
log.Printf("  总订单数: %d", stats["total_orders"])
log.Printf("  待成交: %d", stats["pending_orders"])
log.Printf("  已执行: %d", stats["executed_orders"])
log.Printf("  已失败: %d", stats["failed_orders"])
log.Printf("  黑名单符号: %d", stats["blacklisted_symbols"])

// 按交易对统计
bySymbol := stats["by_symbol"].(map[string]interface{})
for symbol, data := range bySymbol {
    d := data.(map[string]interface{})
    log.Printf("  %s: 待成交=%d, 已执行=%d, 已失败=%d",
               symbol, d["pending"], d["executed"], d["failed"])
}
```

## 配置示例

```toml
# config.toml

[api]
binance_key = "your_binance_api_key"
binance_secret = "your_binance_api_secret"
bybit_key = "your_bybit_api_key"
bybit_secret = "your_bybit_api_secret"

[trading]
# 套利执行配置
min_profit_rate = 0.001  # 最小净利润 0.1%
min_profit_usd = 10      # 最小净利润 10 USD
quantity_per_trade = 1.0 # 每次交易数量
leverage = 1.0           # 杠杆倍数（1=现货）

# 保证金配置
total_margin = 10000.0   # 总保证金 10000 USD
max_single_order = 0.05  # 单笔最多5%保证金
max_concurrent_orders = 5 # 最多5个待成交

# 防重复配置
dedup_window_sec = 5     # 5秒去重
cooldown_period_sec = 10 # 10秒冷却
max_pending_per_symbol = 2 # 单对最多2个待成交
blacklist_ttl_sec = 30   # 黑名单30秒

# 止损配置
stop_loss_rate = 0.05    # 利润下跌5%时止损
```

## 总结

防重复下单系统通过以下层级的防护，确保订单执行的安全性和效率：

1. **防重复层** - 时间窗口、冷却期、待成交限制、黑名单
2. **验证层** - REST API查询实际订单状态
3. **风控层** - 保证金检查、持仓监控、自动止损
4. **恢复层** - 失败隔离、自动重试、状态同步

这三层防护相互配合，形成一个可靠的、可扩展的、易维护的订单执行系统。
