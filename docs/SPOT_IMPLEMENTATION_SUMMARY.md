# Spot 交易支持 - 实施总结

## ✅ 已完成的工作

### 1. 核心组件

#### TradeTypeManager (新增)
```
internal/domain/service/trade_type_manager.go (65 行)
```
- 支持同时管理期货和现货客户端
- `SetFuturesClients()` - 设置期货客户端
- `SetSpotClients()` - 设置现货客户端
- `GetOrderManager(tradeType)` - 获取对应交易类型的订单管理器
- `GetAccountManager(tradeType)` - 获取对应交易类型的账户管理器
- `GetAvailableTradeTypes()` - 获取可用交易类型列表

#### Binance Spot 客户端 (新增)
```
internal/infrastructure/exchange/binance/
├── spot_client.go (30 行)
└── spot_account_client.go (85 行)
```

#### Bybit Spot 客户端 (新增)
```
internal/infrastructure/exchange/bybit/
├── spot_client.go (30 行)
└── spot_account_client.go (85 行)
```

### 2. 现货适配器

在 `cmd/xarb/main.go` 中添加了：
- `binanceSpotOrderClientAdapter` - Binance 现货订单适配器
- `bybitSpotOrderClientAdapter` - Bybit 现货订单适配器

### 3. 初始化逻辑重构

**原来的:**
```go
initializeAPIClients() 
└─ 混合期货和现货初始化逻辑
```

**现在的:**
```go
initializeAPIClients()
├─ initializeFuturesClients()
│  ├─ 创建期货订单客户端
│  ├─ 创建期货账户查询客户端
│  └─ 返回期货 OrderManager + AccountManager
│
└─ initializeSpotClients()
   ├─ 创建现货订单客户端
   ├─ 创建现货账户查询客户端
   └─ 返回现货 OrderManager + AccountManager
```

## 📁 新增文件清单

```
xarb/
├── internal/
│   ├── domain/service/
│   │   └── trade_type_manager.go ✨ (65 行)
│   │
│   └── infrastructure/exchange/
│       ├── binance/
│       │   ├── spot_client.go ✨ (30 行)
│       │   └── spot_account_client.go ✨ (85 行)
│       │
│       └── bybit/
│           ├── spot_client.go ✨ (30 行)
│           └── spot_account_client.go ✨ (85 行)
│
├── SPOT_SUPPORT_DESIGN.md (设计文档)
└── cmd/xarb/main.go (已修改，+120 行现货适配器和初始化逻辑)
```

## 🔄 修改的文件

### 1. cmd/xarb/main.go
```go
// 修改前
type APIClients struct {
    OrderManager      *domainservice.OrderManager
    ArbitrageExecutor *domainservice.ArbitrageExecutor
    AccountManager    *domainservice.AccountManager
}

// 修改后
type APIClients struct {
    TradeTypeManager  *domainservice.TradeTypeManager
    ArbitrageExecutor *domainservice.ArbitrageExecutor
}
```

**新增函数:**
- `initializeFuturesClients()` - 期货初始化
- `initializeSpotClients()` - 现货初始化
- `shouldInitSpotClients()` - 检查是否启用现货
- `newBinanceSpotOrderAdapter()` - Binance 现货适配器工厂
- `newBybitSpotOrderAdapter()` - Bybit 现货适配器工厂

### 2. internal/application/usecase/monitor/service.go
```go
// 添加到 ServiceDeps
type ServiceDeps struct {
    // ... existing fields ...
    TradeTypeManager  *domainservice.TradeTypeManager  // 新增
}
```

## 💡 使用方式

### 方式 1: 按交易类型选择（推荐）

```go
// 在 monitor service 中
func (s *Service) handleArbitrageSignal(ctx, symbol, delta) {
    // 根据 symbol 或配置选择交易类型
    var tradeType string
    if s.isFuturesSymbol(symbol) {
        tradeType = "futures"
    } else {
        tradeType = "spot"
    }
    
    // 获取对应的管理器
    orderMgr, _ := s.deps.TradeTypeManager.GetOrderManager(tradeType)
    accountMgr, _ := s.deps.TradeTypeManager.GetAccountManager(tradeType)
    
    // 使用相应的管理器执行订单
    execution, err := orderMgr.ExecuteArbitrage(ctx, s.deps.Executor, symbol, ...)
}
```

### 方式 2: 一次性获取所有信息

```go
// 获取可用交易类型
types := s.deps.TradeTypeManager.GetAvailableTradeTypes()
// 可能返回: ["futures", "spot"]

// 检查是否支持
hasFutures := s.deps.TradeTypeManager.HasFutures()
hasSpot := s.deps.TradeTypeManager.HasSpot()

if hasFutures && hasSpot {
    log.Info().Msg("Both futures and spot trading enabled")
}
```

## 📋 配置示例

```toml
# config.toml

# 期货配置（Binance Futures + Bybit Perpetual）
[api.binance]
enabled = true
key = "your_binance_futures_api_key"
secret = "your_binance_futures_api_secret"

[api.bybit]
enabled = true
key = "your_bybit_perpetual_api_key"
secret = "your_bybit_perpetual_api_secret"

# 如果来自同一账户，可以使用相同的密钥
# Binance 和 Bybit 都支持在 API key 中配置权限范围
```

## 🚀 快速开始

### 步骤 1: 生成现货 API 密钥

**Binance:**
- 登录 https://www.binance.com/en/account/api-management
- 创建新的 API Key
- 启用权限: `Enable Reading` + `Enable Spot & Margin Trading`
- 复制 API Key 和 Secret Key

**Bybit:**
- 登录 https://www.bybit.com/user-center/account-api
- 创建新的 API Key
- 启用权限: `Spot Trading`
- 复制 API Key 和 Secret Key

### 步骤 2: 配置

如果期货和现货使用同一账户，API 密钥相同：
```toml
[api.binance]
enabled = true
key = "your_api_key"
secret = "your_api_secret"
# 这个密钥同时支持期货和现货
```

### 步骤 3: 在代码中使用

```go
// monitor/service.go 中
func (s *Service) executeSpotTrade(ctx, symbol, delta) {
    // 获取现货管理器
    orderMgr, err := s.deps.TradeTypeManager.GetOrderManager("spot")
    if err != nil {
        log.Warn().Err(err).Msg("Spot trading not available")
        return
    }
    
    // 执行现货订单
    execution, err := orderMgr.ExecuteArbitrage(ctx, s.deps.Executor, symbol, ...)
    if err != nil {
        log.Error().Err(err).Msg("Spot execution failed")
        return
    }
    
    log.Info().
        Str("symbol", symbol).
        Str("type", "spot").
        Msg("Spot arbitrage executed successfully")
}
```

## 📊 架构图

```
                        TradeTypeManager
                              |
                    ____________________
                   |                    |
            FuturesClients         SpotClients
                   |                    |
            (Binance Futures)   (Binance Spot)
            (Bybit Perpetual)   (Bybit Spot)
                   |                    |
            OrderManager         OrderManager
            AccountManager       AccountManager
                   |                    |
                   └────────┬───────────┘
                           |
                    Monitor Service
                      (选择交易类型)
```

## 🔧 实现状态

### 已完成
- ✅ TradeTypeManager 框架
- ✅ Spot 客户端框架（TODO 实现 REST API）
- ✅ Spot 账户客户端框架（TODO 实现 REST API）
- ✅ Spot 订单适配器
- ✅ 初始化逻辑重构
- ✅ Monitor Service 集成
- ✅ 编译验证 (17MB)

### 待完成
- ⏳ Binance Spot: PlaceOrder, CancelOrder, GetOrderStatus 实现
- ⏳ Binance Spot: GetAccount, GetPositions, GetOpenOrders, GetOrderHistory 实现
- ⏳ Bybit Spot: PlaceOrder, CancelOrder, GetOrderStatus 实现
- ⏳ Bybit Spot: GetAccount, GetPositions, GetOpenOrders, GetOrderHistory 实现
- ⏳ 测试（期货和现货分别测试）

## 💾 代码行数统计

| 文件 | 行数 |
|------|------|
| trade_type_manager.go | 65 |
| binance/spot_client.go | 30 |
| binance/spot_account_client.go | 85 |
| bybit/spot_client.go | 30 |
| bybit/spot_account_client.go | 85 |
| main.go (新增代码) | +120 |
| monitor/service.go (修改) | +1 |
| **总计** | **416** |

## 🎯 下一步

### 立即可做
1. 实现 Binance Spot REST API 调用
2. 实现 Bybit Spot REST API 调用
3. 在 Monitor Service 中添加交易类型判断逻辑
4. 测试期货和现货同时运行

### 可选优化
1. 添加配置选项指定哪些币对用期货，哪些用现货
2. 添加风险控制 - 期货和现货分别设置止损
3. 添加手续费差异处理 - 期货和现货手续费不同

## ✨ 特点

✅ **清晰的架构** - 期货和现货完全分离
✅ **灵活切换** - 同一个 symbol 可以根据条件选择交易类型
✅ **代码复用** - 使用同一套 OrderManager/AccountManager 框架
✅ **向后兼容** - 现有期货逻辑不需要改变
✅ **易于扩展** - 添加新交易所只需要实现客户端

## 📝 注意事项

1. **API 权限**
   - Binance: 期货和现货使用不同的权限范围
   - Bybit: 如果同账户，需要权限包含两种
   - 推荐使用不同的 API Key 来分离权限

2. **资金隔离**
   - Binance: 期货和现货账户资金分开
   - Bybit: Linear (永续合约) 和 Spot 账户分开
   - 转账需要手动操作

3. **风险管理**
   - 现货没有杠杆，但有本金风险
   - 期货有爆仓风险
   - 建议分别设置风险参数

## 🆘 调试

### 查看可用交易类型
```go
types := tradeTypeManager.GetAvailableTradeTypes()
fmt.Printf("Available trade types: %v\n", types)
```

### 检查管理器初始化
```go
if tradeTypeManager.HasFutures() {
    log.Info().Msg("Futures managers initialized")
}
if tradeTypeManager.HasSpot() {
    log.Info().Msg("Spot managers initialized")
}
```

### 获取错误信息
```go
orderMgr, err := tradeTypeManager.GetOrderManager("spot")
if err != nil {
    log.Error().Err(err).Msg("Failed to get spot order manager")
}
```

---

**编译状态**: ✅ 成功 (17MB)
**下一步**: 实现 REST API 调用
