# 套利执行引擎使用指南

## 概述

项目现在拥有完整的套利执行框架，可以根据 WebSocket 实时价格数据计算利润，只有当有纯利润时才执行下单。

## 核心组件

### 1. ArbitrageExecutor - 套利分析引擎

**职责**：分析价差、计算成本、决定是否有利可图

```go
executor := service.NewArbitrageExecutor()

// 自定义费用设置
executor.SetFees(
    0.02,   // Binance maker fee 0.02%
    0.04,   // Binance taker fee 0.04%
    0.01,   // Bybit maker fee 0.01%
    0.03,   // Bybit taker fee 0.03%
)

// 自定义融资费率
executor.SetFundingRates(0.001, 0.0008) // Binance, Bybit

// 设置最小利润阈值
executor.SetMinProfitThreshold(0.1) // 最少赚 0.1%

// 设置预计持仓时间（小时）
executor.SetHoldingHours(1)
```

### 2. OpportunityAnalysis - 机会分析结果

```go
// 分析 BTC 价差机会
analysis := executor.AnalyzeOpportunity(
    "BTCUSDT",      // 交易对
    45000.0,        // Binance 价格
    45100.0,        // Bybit 价格
    1.0,            // 交易数量
)

// 检查结果
if analysis.IsOpportunity {
    fmt.Printf("机会! 净利润: %.4f%% ($%.2f)\n", 
        analysis.NetProfitRate, 
        analysis.NetProfitUSD,
    )
    fmt.Printf("原因: %s\n", analysis.Reason)
} else {
    fmt.Printf("无利可图: %s\n", analysis.Reason)
}
```

### 3. OrderManager - 订单执行管理

**职责**：实际下单、订单跟踪、风险控制

```go
// 创建订单管理器
orderManager := service.NewOrderManager(binanceClient, bybitClient)

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
    log.Error().Err(err).Msg("交易执行失败")
    return
}

log.Info().
    Str("direction", execution.Direction).
    Float64("profit", execution.ExpectedProfit).
    Msg("套利交易执行成功")
```

## 成本分析详解

### 交易费用 (Trading Fee)
- **Binance Maker**: 0.02% (挂单)
- **Binance Taker**: 0.04% (吃单)
- **Bybit Maker**: 0.01% (挂单)
- **Bybit Taker**: 0.03% (吃单)

### 融资费率 (Funding Fee)
- **Binance**: 约 0.1% 每 8 小时
- **Bybit**: 约 0.08% 每 8 小时
- **计算**: (费率 × 持仓小时数 / 8)

### 总成本公式

```
总成本比例 = 交易手续费 + 融资费率
         = (Maker1% + Taker2%) + (FundingRate1% + FundingRate2%)

例如 1 小时持仓:
BUY_BINANCE_SELL_BYBIT 的成本 = 0.04% + 0.01% + (0.001 + 0.0008) * 100 / 8
                           = 0.05% + 0.0225%
                           = 0.0725%
```

## 交易流程

### 场景 1: Bybit 价格 > Binance 价格

**策略**: 在 Binance 买入，在 Bybit 卖出

```
步骤 1: 分析价差
  ├─ BinancePrice: 45000.0
  ├─ BybitPrice:   45100.0
  ├─ Spread:       100.0 (0.222%)
  └─ NetProfit:    需要 >= 0.1% 才下单

步骤 2: 计算成本
  ├─ Binance 吃单费: 0.04%
  ├─ Bybit 挂单费:  0.01%
  ├─ 融资费 (1h):   0.0225%
  └─ 总成本:        0.0725%

步骤 3: 计算净利润
  ├─ 毛利: 0.222%
  ├─ 成本: 0.0725%
  └─ 净利: 0.1495% ✓ (> 0.1% 阈值)

步骤 4: 下单
  ├─ Binance BUY  1 BTC @ 市价
  ├─ 等待成交
  └─ Bybit SELL 1 BTC @ 市价
```

### 场景 2: Binance 价格 > Bybit 价格

**策略**: 在 Bybit 买入，在 Binance 卖出

```
步骤 1: 分析价差
  ├─ BybitPrice:   44900.0
  ├─ BinancePrice: 45000.0
  ├─ Spread:       100.0 (0.223%)
  └─ NetProfit:    计算...

步骤 2: 计算成本
  ├─ Bybit 吃单费:  0.03%
  ├─ Binance 挂单费: 0.02%
  ├─ 融资费:        0.0225%
  └─ 总成本:        0.0725%

步骤 3: 计算净利润
  └─ 净利: 0.1505% ✓

步骤 4: 下单
  ├─ Bybit BUY 1 BTC @ 市价
  ├─ 等待成交
  └─ Binance SELL 1 BTC @ 市价
```

## 集成到监控服务

在 Monitor Service 的主循环中添加套利执行逻辑：

```go
// 在 monitor/service.go Run() 方法中
executor := domainservice.NewArbitrageExecutor()
executor.SetMinProfitThreshold(0.15) // 设置利润阈值 0.15%

// 在接收价格更新时
case t := <-merged:
    changed := s.st.Apply(t)
    if changed {
        // ... 现有逻辑 ...
        
        // 新增：套利分析
        binancePrice := s.st.LatestPrice("BINANCE", "BTCUSDT")
        bybitPrice := s.st.LatestPrice("BYBIT", "BTCUSDT")
        
        if binancePrice > 0 && bybitPrice > 0 {
            analysis := executor.AnalyzeOpportunity(
                "BTCUSDT",
                binancePrice,
                bybitPrice,
                1.0, // 交易数量
            )
            
            // 只有有利可图才会执行
            if analysis.IsOpportunity {
                log.Warn().
                    Str("symbol", "BTCUSDT").
                    Str("direction", analysis.Reason).
                    Float64("profit", analysis.NetProfitUSD).
                    Msg("套利信号")
                
                // 可在此触发下单
                // err := orderManager.ExecuteArbitrage(...)
            }
        }
    }
```

## 配置建议

### 激进策略 (高风险高收益)
```go
executor.SetMinProfitThreshold(0.05)  // 仅需 0.05% 利润
executor.SetHoldingHours(0.5)         // 快速平仓
```

### 保守策略 (低风险)
```go
executor.SetMinProfitThreshold(0.25)  // 要求 0.25% 利润
executor.SetHoldingHours(4)           // 长期持仓
```

### 平衡策略 (推荐)
```go
executor.SetMinProfitThreshold(0.15)  // 0.15% 利润
executor.SetHoldingHours(1)           // 1 小时持仓
```

## 风险提示

1. **滑点风险**: 市价单成交价可能偏离期望
2. **融资费波动**: 融资费率不是固定的，可能变化
3. **执行延迟**: 网络延迟可能导致价差消失
4. **流动性风险**: 大额订单可能无法完全成交
5. **对手风险**: 交易所宕机或卡顿

## 下一步

1. 实现 REST API 客户端 (OrderClient 接口)
2. 接入真实 Binance 和 Bybit 账户
3. 添加订单验证和风险检查
4. 实现 WebSocket 订单更新推送
5. 添加止损和风险控制逻辑
