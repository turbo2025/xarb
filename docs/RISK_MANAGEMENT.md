# 风险管理和自动平仓系统

## 概述

系统现在拥有完整的风险控制框架，确保所有交易都有保证金保护，并通过自动平仓机制确保不亏损。

## 核心特性

### 1. 保证金管理 (MarginManager)

**职责**：
- ✅ 验证保证金是否足够
- ✅ 限制单笔订单不超过总保证金 5%
- ✅ 限制最多 5 个并发订单
- ✅ 追踪已实现盈亏
- ✅ 评估风险等级

### 2. 平仓策略

#### 策略 A: 挂单平仓 (Limit Close)
- 设置一个高于预期的限价单
- 等待市场到达目标价格
- 适合：流动性好的交易对

#### 策略 B: 市价平仓 (Market Close)
- 利润下跌到历史最高利润 × 95% 时触发
- 确保立即成交，不继续亏损
- 适合：避免风险扩大

## 集成示例

### 初始化

```go
import (
    "xarb/internal/domain/service"
)

// 1. 创建保证金管理器（初始 10000 USD）
marginMgr := service.NewMarginManager(10000)

// 2. 自定义保证金参数
marginMgr.MaxMarginPerOrder = 0.05      // 单笔最多 5%
marginMgr.MaxOrderCount = 5             // 最多 5 个订单
marginMgr.StopLossProfitRate = 0.05     // 利润跌 5% 就止损

// 3. 创建订单管理器
orderManager := service.NewOrderManager(binanceClient, bybitClient)
orderManager.SetMarginManager(marginMgr)

// 4. 设置杠杆（现货为 1，合约可设置更高）
orderManager.Leverage = 1.0
```

### 执行交易（自动保证金检查）

```go
// 执行套利交易
execution, err := orderManager.ExecuteArbitrage(
    ctx,
    executor,
    "BTCUSDT",
    45000.0,  // Binance 价格
    45100.0,  // Bybit 价格
    1.0,      // 交易数量
)

if err != nil {
    // 可能的错误原因：
    // 1. "max order count reached" - 已有 5 个订单
    // 2. "order margin exceeds limit" - 保证金不足
    // 3. "insufficient margin" - 可用保证金不足
    log.Error().Err(err).Msg("交易执行失败")
    return
}

log.Info().
    Str("direction", execution.Direction).
    Float64("profit", execution.ExpectedProfit).
    Msg("交易执行成功")
```

### 监控和自动平仓

```go
// 定期调用此函数（例如每秒）来监控持仓
func monitorLoop(ctx context.Context, orderManager *service.OrderManager) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // 从 WebSocket 获取最新价格
            priceFeeds := map[string][2]float64{
                "BTCUSDT": {45000.0, 45100.0}, // [Binance, Bybit]
                "ETHUSDT": {2500.0, 2510.0},
            }

            // 监控所有持仓，自动平仓
            if err := orderManager.MonitorAndClosePositions(ctx, priceFeeds); err != nil {
                log.Error().Err(err).Msg("监控平仓出错")
            }

            // 显示账户状态
            status := orderManager.GetAccountStatus()
            fmt.Printf("[Account] Margin: %.2f/%.2f USD (%.1f%%) | " +
                "Orders: %d | Positions: %d | PnL: %.2f USD | Risk: %s\n",
                status.UsedMargin,
                status.TotalMargin,
                status.MarginUsageRate,
                status.ActiveOrderCount,
                status.OpenPositionCount,
                status.RealizedPnL,
                status.RiskLevel,
            )
        }
    }
}
```

## 平仓流程详解

### 场景 1: 利润下跌触发市价平仓

```
时间线：
┌──────────────────────────────────┐
│ 1. 执行交易                       │
│    买入: $45000                   │
│    卖出: $45100                   │
│    期望利润: $100 (0.22%)         │
└──────────────────────────────────┘
                ↓
┌──────────────────────────────────┐
│ 2. 持仓 1 秒后                    │
│    Binance: $45000 (不变)        │
│    Bybit:   $45080 (跌了 20)     │
│    当前利润: $80 (0.18%)          │
│    历史最高: $100                 │
│    下跌幅度: 20% (> 5%)           │
│    → 触发止损                     │
└──────────────────────────────────┘
                ↓
┌──────────────────────────────────┐
│ 3. 市价平仓                       │
│    立即以 $45080 卖出             │
│    实现利润: $80 USD              │
│    风险状态: 安全 ✓              │
└──────────────────────────────────┘
```

### 场景 2: 主动挂单平仓

```go
// 在持仓初期设置挂单平仓目标
// 等待市场到达目标价格

err := orderManager.TryLimitClose(
    ctx,
    "BTCUSDT",
    45150.0, // 设置限价卖出价格（高于预期）
)

// 订单状态：
// - 发出: 以 45150 限价单挂单
// - 成交: 市场触及 45150，自动成交
// - 不成交: 定期检查，如果利润跌到 5% 就市价平仓
```

## 保证金计算示例

### 单笔订单保证金检查

```
交易对: BTCUSDT
数量: 1 BTC
Binance 价格: $45000

计算保证金需求:
现货模式 (杠杆 = 1):
  保证金 = (价格 × 数量) / 杠杆
         = ($45000 × 1) / 1
         = $45000

账户保证金: $10000
单笔限制 (5%): $10000 × 5% = $500

对比:
  需求 $45000 > 限制 $500 ❌ 不允许

解决方案:
  - 增加账户保证金
  - 减少单笔交易数量
  - 使用杠杆（风险更高）
```

### 保证金使用率等级

```
使用率 (0-40%)      → "LOW"      - 可以继续开仓
使用率 (40-60%)     → "MEDIUM"   - 谨慎开仓
使用率 (60-80%)     → "HIGH"     - 停止开仓，准备平仓
使用率 (80%+)       → "CRITICAL" - 紧急平仓风险
```

## 自动平仓触发条件

### 条件 1: 利润下跌 5%

```go
if 当前利润 < 历史最高利润 × 0.95 {
    立即市价平仓()
}

示例:
历史最高利润: $100
下跌 5% 阈值: $95
当前利润: $94 → 触发市价平仓
```

### 条件 2: 保证金不足

```go
if 可用保证金 < 0 {
    强制平仓所有头寸()
}
```

### 条件 3: 订单超时

```go
if 订单挂单时间 > 10分钟 && 未成交 {
    撤销挂单()
    市价平仓()
}
```

## API 参考

### MarginManager

```go
// 检查订单是否可执行
err := marginMgr.CanExecuteOrder(symbol, requiredMargin)

// 注册新订单
err := marginMgr.RegisterOrder(
    orderID, symbol, direction,
    quantity, marginUsed, expectedProfit,
)

// 订单成交后更新
err := marginMgr.ExecuteOrder(
    orderID, buyOrderID, sellOrderID,
    symbol, quantity, buyPrice, sellPrice,
)

// 更新实时利润（每收到新价格都应调用）
pos, err := marginMgr.UpdatePositionProfit(
    symbol, currentBuyPrice, currentSellPrice,
)

// 检查是否需要止损
needStop, pos, err := marginMgr.NeedStopLoss(symbol)
if needStop {
    // 立即市价平仓
}

// 获取账户状态
status := marginMgr.GetMarginStatus()
fmt.Printf("Margin Usage: %.1f%% (%d active orders)\n",
    status.UsageRate, status.ActiveOrderCount)
```

### OrderManager

```go
// 设置保证金管理器
orderManager.SetMarginManager(marginMgr)

// 执行交易（自动检查保证金）
execution, err := orderManager.ExecuteArbitrage(
    ctx, executor, symbol, binancePrice, bybitPrice, quantity,
)

// 监控和自动平仓
err := orderManager.MonitorAndClosePositions(ctx, priceFeeds)

// 主动挂单平仓
err := orderManager.TryLimitClose(ctx, symbol, limitPrice)

// 获取账户状态
status := orderManager.GetAccountStatus()
fmt.Printf("Risk Level: %s, Open Positions: %d\n",
    status.RiskLevel, status.OpenPositionCount)
```

## 风险提示

1. **流动性风险**: 大额市价单可能无法完全成交
2. **时间风险**: 从检测到平仓可能有延迟，利润继续下跌
3. **网络风险**: 连接中断可能导致持仓无人看管
4. **滑点风险**: 市价单实际成交价可能远低于预期
5. **融资费风险**: 融资费率可能急剧变化（极端行情）

## 最佳实践

### 1. 设置合理的保证金
```go
// 保守: 大额账户，风险低
marginMgr := service.NewMarginManager(100000) // 10 万 USD
marginMgr.MaxMarginPerOrder = 0.02            // 单笔 2%
marginMgr.MaxOrderCount = 3                   // 最多 3 个

// 激进: 小额账户，追求高收益
marginMgr := service.NewMarginManager(1000)   // 1000 USD
marginMgr.MaxMarginPerOrder = 0.10            // 单笔 10%
marginMgr.MaxOrderCount = 5                   // 最多 5 个
```

### 2. 定期监控（至少每秒）
```go
// 启用自动监控
monitorTicker := time.NewTicker(1 * time.Second)
for range monitorTicker.C {
    orderManager.MonitorAndClosePositions(ctx, latestPrices)
}
```

### 3. 主动平仓策略
```go
// 在交易执行后立即设置挂单平仓目标
if execution.IsSuccess {
    targetPrice := execution.SellPrice * 1.0002 // 目标价 +0.02%
    orderManager.TryLimitClose(ctx, symbol, targetPrice)
}
```

### 4. 实时告警
```go
status := orderManager.GetAccountStatus()
if status.RiskLevel == "HIGH" {
    log.Warn().Msg("⚠️  High risk level, consider closing positions")
}
if status.RiskLevel == "CRITICAL" {
    log.Error().Msg("🚨 CRITICAL risk! Force closing all positions")
    forceCloseAll(ctx)
}
```

## 下一步

1. ✅ 保证金验证 (已完成)
2. ✅ 自动市价平仓 (已完成)
3. ✅ 挂单平仓机制 (已完成)
4. ⏳ WebSocket 订单推送 (待实现)
5. ⏳ 清算价格预测 (待实现)
6. ⏳ 融资费实时跟踪 (待实现)
7. ⏳ 动态杠杆调整 (待实现)
