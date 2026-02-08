# REST API 集成和下单指南

## 概述

项目现在支持完整的 REST API 客户端，可以：
- ✅ 通过 API 下单（下市价单和限价单）
- ✅ 查询持仓和账户信息
- ✅ 查询订单状态
- ✅ 获取融资费率
- ✅ 自动签名和认证

## API 下单 vs WebSocket 下单

| 方面 | REST API | WebSocket |
|------|---------|----------|
| **用途** | **下单、持仓查询、账户管理** | 行情推送（实时价格）|
| **可靠性** | ⭐⭐⭐⭐⭐ 高 | ⭐⭐⭐ 一般 |
| **成交确认** | 同步响应 | 异步推送 |
| **支持功能** | 完整（下单、撤单、查询等）| 行情数据 |
| **实现状态** | ✅ 完成 | ✅ 已有 |

## 推荐方案：混合模式

```
┌─────────────────────────────────────────┐
│ WebSocket (行情推送)                    │
│ ✅ 实时 Binance 价格 (fstream)         │
│ ✅ 实时 Bybit 价格 (stream)            │
│ → 供 ArbitrageExecutor 分析             │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│ REST API (下单执行)                     │
│ ✅ Binance 期货 (/fapi/)               │
│ ✅ Bybit 线性永续 (/v5/)               │
│ → 执行套利交易                          │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│ REST API (定期查询)                     │
│ ✅ 账户信息 (保证金、净值)              │
│ ✅ 开仓持仓 (数量、价格、利润)          │
│ ✅ 订单状态 (成交、待成交)              │
│ → 监控风险和自动平仓                    │
└─────────────────────────────────────────┘
```

## 初始化和集成

### 1. 从配置中加载 API Key

```go
import (
    "xarb/internal/infrastructure/config"
    "xarb/internal/infrastructure/exchange/binance"
    "xarb/internal/infrastructure/exchange/bybit"
)

// 加载配置
cfg, _ := config.LoadConfig("./configs/config.toml")

// 创建 REST 客户端
binanceOrderClient := binance.NewFuturesOrderClient(
    cfg.Exchange.Binance.ApiKey,
    cfg.Exchange.Binance.SecretKey,
)

bybitOrderClient := bybit.NewLinearOrderClient(
    cfg.Exchange.Bybit.ApiKey,
    cfg.Exchange.Bybit.SecretKey,
)
```

### 2. 初始化订单管理器

```go
import (
    "xarb/internal/domain/service"
)

// 创建订单管理器
orderManager := service.NewOrderManager(binanceOrderClient, bybitOrderClient)

// 设置保证金管理器
marginMgr := service.NewMarginManager(10000) // 10000 USD
orderManager.SetMarginManager(marginMgr)

// 从账户查询实时保证金
account, _ := binanceOrderClient.GetAccount(ctx)
marginMgr.TotalMargin = account.TotalWalletBalance
marginMgr.AvailableBalance = account.AvailableBalance
```

## 使用示例

### 示例 1: 下单（自动保证金检查）

```go
// 分析套利机会
executor := domainservice.NewArbitrageExecutor()

// 执行交易（会自动检查保证金）
execution, err := orderManager.ExecuteArbitrage(
    ctx,
    executor,
    "BTCUSDT",
    45000.0,  // Binance 价格
    45100.0,  // Bybit 价格
    1.0,      // 交易数量
)

if err != nil {
    // 可能的错误：
    // - "max order count reached" - 已有 5 个订单
    // - "order margin exceeds limit" - 保证金占比超过 5%
    // - "insufficient margin" - 可用保证金不足
    log.Error().Err(err).Msg("交易执行失败")
    return
}

log.Info().
    Str("buyOrderID", execution.BuyOrderID).
    Str("sellOrderID", execution.SellOrderID).
    Float64("expectedProfit", execution.ExpectedProfit).
    Msg("交易执行成功")
```

### 示例 2: 查询账户信息和持仓

```go
// 查询 Binance 账户
binanceAccount, _ := binanceOrderClient.GetAccount(ctx)
fmt.Printf("Binance 账户:\n")
fmt.Printf("  总余额: %.2f USD\n", binanceAccount.TotalWalletBalance)
fmt.Printf("  可用: %.2f USD\n", binanceAccount.AvailableBalance)
fmt.Printf("  已用保证金: %.2f USD\n", binanceAccount.TotalMarginRequired)
fmt.Printf("  开仓持仓数: %d\n", len(binanceAccount.Positions))

for _, pos := range binanceAccount.Positions {
    fmt.Printf("    - %s: %.4f 张 @ %.2f (未实现: $%.2f)\n",
        pos.Symbol, pos.PositionAmount, pos.EntryPrice, pos.UnrealizedProfit)
}

// 查询 Bybit 开仓持仓
bybitPositions, _ := bybitOrderClient.GetOpenPositions(ctx)
fmt.Printf("Bybit 开仓持仓数: %d\n", len(bybitPositions))
```

### 示例 3: 查询订单状态

```go
// 获取订单状态
status, _ := binanceOrderClient.GetOrderStatus(ctx, "BTCUSDT", orderID)

fmt.Printf("订单 %s 状态:\n", orderID)
fmt.Printf("  状态: %s\n", status.Status)           // NEW, PARTIALLY_FILLED, FILLED
fmt.Printf("  成交: %.8g / %.8g\n", status.ExecutedQuantity, status.Quantity)
fmt.Printf("  平均价: %.2f\n", status.AvgExecutedPrice)

// 根据状态决定下一步
if status.Status == "FILLED" {
    // 订单已完全成交，继续下一步
    fmt.Println("✓ 订单已成交，可以继续进行对冲")
} else if status.ExecutedQuantity > 0 {
    // 部分成交，需要处理剩余
    fmt.Printf("⚠️  部分成交 %.8g，剩余 %.8g\n",
        status.ExecutedQuantity,
        status.Quantity - status.ExecutedQuantity)
}
```

### 示例 4: 定期监控账户和平仓

```go
// 定期任务（每 5 秒）
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

for range ticker.C {
    // 1. 查询账户信息
    account, err := binanceOrderClient.GetAccount(ctx)
    if err != nil {
        log.Error().Err(err).Msg("查询账户失败")
        continue
    }

    // 2. 更新风险管理器
    marginMgr.TotalMargin = account.TotalWalletBalance
    marginMgr.AvailableBalance = account.AvailableBalance

    // 3. 显示账户状态
    status := orderManager.GetAccountStatus()
    fmt.Printf("[%s] Margin: %.1f%% | Orders: %d | PnL: $%.2f | Risk: %s\n",
        time.Now().Format("15:04:05"),
        status.MarginUsageRate,
        status.ActiveOrderCount,
        status.RealizedPnL,
        status.RiskLevel,
    )

    // 4. 监控持仓（自动平仓）
    priceFeeds := map[string][2]float64{
        "BTCUSDT": {45000.0, 45100.0},
        "ETHUSDT": {2500.0, 2510.0},
    }
    orderManager.MonitorAndClosePositions(ctx, priceFeeds)
}
```

### 示例 5: 手动撤单

```go
// 撤销 Binance 订单
err := binanceOrderClient.CancelOrder(ctx, "BTCUSDT", orderID)
if err != nil {
    log.Error().Err(err).Msg("撤销订单失败")
}

// 撤销 Bybit 订单
err = bybitOrderClient.CancelOrder(ctx, "BTCUSDT", orderID)
if err != nil {
    log.Error().Err(err).Msg("撤销订单失败")
}
```

## API 参考

### Binance FuturesOrderClient

```go
// 下单
orderID, err := client.PlaceOrder(ctx, "BTCUSDT", "BUY", 1.0, 45000, false)

// 撤单
err := client.CancelOrder(ctx, "BTCUSDT", orderID)

// 查询订单
status, err := client.GetOrderStatus(ctx, "BTCUSDT", orderID)

// 查询融资费率
fundingRate, err := client.GetFundingRate(ctx, "BTCUSDT")

// 查询账户
account, err := client.GetAccount(ctx)
// account.TotalWalletBalance
// account.AvailableBalance
// account.TotalMarginRequired
// account.Positions

// 查询开仓持仓
positions, err := client.GetOpenPositions(ctx)
```

### Bybit LinearOrderClient

```go
// 下单
orderID, err := client.PlaceOrder(ctx, "BTCUSDT", "Buy", 1.0, 45000, false)

// 撤单
err := client.CancelOrder(ctx, "BTCUSDT", orderID)

// 查询订单
status, err := client.GetOrderStatus(ctx, "BTCUSDT", orderID)

// 查询融资费率
fundingRate, err := client.GetFundingRate(ctx, "BTCUSDT")

// 查询账户
account, err := client.GetAccount(ctx)

// 查询开仓持仓
positions, err := client.GetOpenPositions(ctx)
```

## 订单类型

### 市价单（Market Order）

```go
// 立即以市价成交
orderID, err := binanceOrderClient.PlaceOrder(
    ctx,
    "BTCUSDT",
    "BUY",
    1.0,
    0,           // 价格无关（市价）
    true,        // isMarket = true
)
```

优点：
- ✅ 立即成交
- ✅ 适合套利交易（需要快速成交）

缺点：
- ❌ 可能有滑点（实际成交价格偏离市价）

### 限价单（Limit Order）

```go
// 设置限价，等待成交
orderID, err := binanceOrderClient.PlaceOrder(
    ctx,
    "BTCUSDT",
    "BUY",
    1.0,
    45000.0,     // 具体价格
    false,       // isMarket = false
)
```

优点：
- ✅ 避免滑点
- ✅ 适合平仓时设置目标价格

缺点：
- ❌ 可能不成交
- ❌ 需要定期检查和超时处理

## 风险管理集成

```go
// 创建保证金管理器
marginMgr := service.NewMarginManager(10000)

// 从实盘账户同步初始保证金
account, _ := binanceOrderClient.GetAccount(ctx)
marginMgr.TotalMargin = account.TotalWalletBalance

// 设置风险参数
marginMgr.MaxMarginPerOrder = 0.05      // 单笔最多 5%
marginMgr.MaxOrderCount = 5             // 最多 5 个订单
marginMgr.StopLossProfitRate = 0.05     // 利润跌 5% 止损

// 订单管理器集成
orderManager.SetMarginManager(marginMgr)

// 执行时会自动检查
execution, err := orderManager.ExecuteArbitrage(...)
// ↑ 如果保证金不足会返回错误
```

## 错误处理

```go
execution, err := orderManager.ExecuteArbitrage(...)

if err != nil {
    switch {
    case strings.Contains(err.Error(), "max order count"):
        log.Warn().Msg("已达订单上限，等待平仓")
        
    case strings.Contains(err.Error(), "exceeds limit"):
        log.Warn().Msg("单笔订单过大，需要减少数量")
        
    case strings.Contains(err.Error(), "insufficient margin"):
        log.Error().Msg("保证金不足！需要充值或平仓")
        
    case strings.Contains(err.Error(), "http 429"):
        log.Warn().Msg("请求过频繁，等待重试")
        time.Sleep(time.Second)
        
    default:
        log.Error().Err(err).Msg("未知错误")
    }
    return
}
```

## 性能建议

### 1. 批量查询持仓
```go
// ❌ 不要循环查询每个持仓
for _, symbol := range symbols {
    pos, _ := client.GetOrderStatus(ctx, symbol, ...)
}

// ✅ 一次查询所有持仓
account, _ := client.GetAccount(ctx)
for _, pos := range account.Positions {
    // 处理持仓
}
```

### 2. 缓存融资费率
```go
// 融资费率每 8 小时更新一次，不需要频繁查询
fundingRateCache := make(map[string]float64)
lastUpdate := time.Now()

// 每小时更新一次
if time.Since(lastUpdate) > time.Hour {
    for _, symbol := range symbols {
        rate, _ := client.GetFundingRate(ctx, symbol)
        fundingRateCache[symbol] = rate
    }
    lastUpdate = time.Now()
}
```

### 3. 异步执行
```go
// 不要阻塞主循环
go func() {
    account, _ := client.GetAccount(ctx)
    // 处理账户数据
}()
```

## 下一步

- ✅ REST API 客户端（已完成）
- ✅ 下单和撤单（已完成）
- ✅ 持仓和账户查询（已完成）
- ✅ 自动签名和认证（已完成）
- ⏳ WebSocket 订单推送（可选优化）
- ⏳ 更多交易所支持（OKX、Bitget）
- ⏳ 批量操作优化
