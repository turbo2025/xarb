# 支持 Spot 客户端的设计方案

## 问题分析

当前系统只支持期货（Futures）：
- Binance: `FuturesOrderClient` (期货)
- Bybit: `LinearOrderClient` (永续合约)

如果要支持现货（Spot），有几个设计方案：

## 方案对比

### 方案 1: 分离的管理器（推荐）
```go
// 期货管理器
futuresOrderManager := NewOrderManager(binanceFuturesClient, bybitLinearClient)

// 现货管理器
spotOrderManager := NewOrderManager(binanceSpotClient, bybitSpotClient)

// 在 monitor service 中根据交易类型选择
if tradeType == "futures" {
    futuresOrderManager.ExecuteArbitrage(...)
} else {
    spotOrderManager.ExecuteArbitrage(...)
}
```

**优点:**
- 清晰、易维护
- 不同策略可以用不同的管理器
- 期货和现货逻辑完全独立

**缺点:**
- 需要维护两个管理器

---

### 方案 2: 账户类型前缀（现在采用）
```go
// 注册时用前缀区分
accountManager.RegisterClient("binance-futures", binanceFuturesAccountClient)
accountManager.RegisterClient("binance-spot", binanceSpotAccountClient)

// 查询时指定账户类型
account, _ := accountManager.GetAccount(ctx, "binance-futures")
account, _ := accountManager.GetAccount(ctx, "binance-spot")
```

**优点:**
- 单个管理器支持多种账户类型
- 灵活、可扩展

**缺点:**
- 需要显式指定账户类型

---

## 实施方案

### 第一步: 创建 Spot 客户端

#### Binance Spot
```go
// internal/infrastructure/exchange/binance/spot_client.go
type SpotOrderClient struct {
    apiKey    string
    apiSecret string
    client    *http.Client
}

func NewSpotOrderClient(apiKey, apiSecret string) *SpotOrderClient {
    return &SpotOrderClient{
        apiKey:    apiKey,
        apiSecret: apiSecret,
        client:    &http.Client{Timeout: 10 * time.Second},
    }
}

// 实现 OrderClient 接口
func (c *SpotOrderClient) PlaceOrder(ctx, symbol, side, quantity, price, isMarket) (string, error)
func (c *SpotOrderClient) CancelOrder(ctx, symbol, orderId) error
func (c *SpotOrderClient) GetOrderStatus(ctx, symbol, orderId) (*OrderStatus, error)
func (c *SpotOrderClient) GetFundingRate(ctx, symbol) (float64, error)  // spot 不适用
```

#### Bybit Spot
```go
// internal/infrastructure/exchange/bybit/spot_client.go
type SpotOrderClient struct {
    apiKey    string
    apiSecret string
    client    *http.Client
}

func NewSpotOrderClient(apiKey, apiSecret string) *SpotOrderClient {
    // 类似实现
}
```

### 第二步: 修改配置

```toml
# config.toml

[api.binance]
enabled = true
key = "..."
secret = "..."

[api.binance_spot]  # 新增
enabled = false     # 默认关闭
key = "..."
secret = "..."

[api.bybit]
enabled = true
key = "..."
secret = "..."

[api.bybit_spot]    # 新增
enabled = false
key = "..."
secret = "..."
```

### 第三步: 更新初始化逻辑

```go
// cmd/xarb/main.go

type APIClients struct {
    // 期货管理器
    FuturesOrderManager *domainservice.OrderManager
    FuturesAccountManager *domainservice.AccountManager
    
    // 现货管理器（可选）
    SpotOrderManager *domainservice.OrderManager
    SpotAccountManager *domainservice.AccountManager
    
    ArbitrageExecutor *domainservice.ArbitrageExecutor
}

func initializeAPIClients(cfg *config.Config) *APIClients {
    clients := &APIClients{}
    
    // 初始化期货客户端
    clients.FuturesOrderManager, clients.FuturesAccountManager = 
        initializeFuturesClients(cfg)
    
    // 初始化现货客户端（如果启用）
    if shouldInitSpotClients(cfg) {
        clients.SpotOrderManager, clients.SpotAccountManager = 
            initializeSpotClients(cfg)
    }
    
    clients.ArbitrageExecutor = domainservice.NewArbitrageExecutor()
    return clients
}

func initializeFuturesClients(cfg *config.Config) (
    *domainservice.OrderManager,
    *domainservice.AccountManager) {
    
    accountManager := domainservice.NewAccountManager(5 * time.Second)
    
    binanceOrderClient := binance.NewFuturesOrderClient(...)
    bybitOrderClient := bybit.NewLinearOrderClient(...)
    binanceAccountClient := binance.NewBinanceAccountClient(...)
    bybitAccountClient := bybit.NewBybitAccountClient(...)
    
    accountManager.RegisterClient("binance", binanceAccountClient)
    accountManager.RegisterClient("bybit", bybitAccountClient)
    
    orderManager := domainservice.NewOrderManager(
        newBinanceOrderAdapter(binanceOrderClient),
        newBybitOrderAdapter(bybitOrderClient),
    )
    
    log.Info().Msg("✓ Futures REST API clients initialized")
    return orderManager, accountManager
}

func initializeSpotClients(cfg *config.Config) (
    *domainservice.OrderManager,
    *domainservice.AccountManager) {
    
    accountManager := domainservice.NewAccountManager(5 * time.Second)
    
    binanceOrderClient := binance.NewSpotOrderClient(...)
    bybitOrderClient := bybit.NewSpotOrderClient(...)
    binanceAccountClient := binance.NewBinanceSpotAccountClient(...)
    bybitAccountClient := bybit.NewBybitSpotAccountClient(...)
    
    accountManager.RegisterClient("binance", binanceAccountClient)
    accountManager.RegisterClient("bybit", bybitAccountClient)
    
    orderManager := domainservice.NewOrderManager(
        newBinanceSpotOrderAdapter(binanceOrderClient),
        newBybitSpotOrderAdapter(bybitOrderClient),
    )
    
    log.Info().Msg("✓ Spot REST API clients initialized")
    return orderManager, accountManager
}

func shouldInitSpotClients(cfg *config.Config) bool {
    return (cfg.API.BinanceSpot.Enabled || cfg.API.BybitSpot.Enabled)
}
```

### 第四步: 在 Monitor Service 中使用

```go
// internal/application/usecase/monitor/service.go

type ServiceDeps struct {
    // ... existing fields ...
    
    // 期货
    OrderManager *domainservice.OrderManager
    AccountManager *domainservice.AccountManager
    
    // 现货（可选）
    SpotOrderManager *domainservice.OrderManager
    SpotAccountManager *domainservice.AccountManager
}

type Service struct {
    // ... existing fields ...
    tradeType string  // "futures" or "spot"
}

func (s *Service) handleArbitrageSignal(ctx, symbol, delta) {
    var orderMgr *domainservice.OrderManager
    var accountMgr *domainservice.AccountManager
    
    if s.tradeType == "futures" {
        orderMgr = s.deps.OrderManager
        accountMgr = s.deps.AccountManager
    } else {
        orderMgr = s.deps.SpotOrderManager
        accountMgr = s.deps.SpotAccountManager
    }
    
    // 使用相应的管理器执行订单
    execution, err := orderMgr.ExecuteArbitrage(ctx, s.deps.Executor, symbol, ...)
    
    if err == nil {
        // 验证订单
        s.verifyOrderExecution(ctx, symbol, execution, accountMgr)
    }
}
```

---

## 更简洁的方案：通用适配器

如果想保持 ServiceDeps 不变，可以创建一个 TradeTypeManager：

```go
type TradeTypeManager struct {
    futures struct {
        orderManager   *domainservice.OrderManager
        accountManager *domainservice.AccountManager
    }
    spot struct {
        orderManager   *domainservice.OrderManager
        accountManager *domainservice.AccountManager
    }
}

func (tm *TradeTypeManager) GetOrderManager(tradeType string) *domainservice.OrderManager {
    if tradeType == "spot" {
        return tm.spot.orderManager
    }
    return tm.futures.orderManager
}

func (tm *TradeTypeManager) GetAccountManager(tradeType string) *domainservice.AccountManager {
    if tradeType == "spot" {
        return tm.spot.accountManager
    }
    return tm.futures.accountManager
}

// 在 main.go 中
type APIClients struct {
    TradeTypeManager *TradeTypeManager
    ArbitrageExecutor *domainservice.ArbitrageExecutor
}

// 在 service 中
orderMgr := s.deps.TradeTypeManager.GetOrderManager(s.tradeType)
accountMgr := s.deps.TradeTypeManager.GetAccountManager(s.tradeType)
```

---

## 完整代码变更清单

### 需要创建的新文件
```
internal/infrastructure/exchange/binance/
├── spot_client.go                 (新)
└── spot_account_client.go         (新)

internal/infrastructure/exchange/bybit/
├── spot_client.go                 (新)
└── spot_account_client.go         (新)

internal/domain/service/
└── trade_type_manager.go          (可选，用于简化)
```

### 需要修改的文件
```
cmd/xarb/main.go
├── 修改 APIClients 结构体
├── 修改 initializeAPIClients() 函数
├── 添加 initializeFuturesClients()
├── 添加 initializeSpotClients()
└── 添加适配器工厂函数

internal/application/usecase/monitor/service.go
├── 修改 ServiceDeps
├── 添加 tradeType 字段
└── 更新 handleArbitrageSignal()

internal/infrastructure/config/config.go
├── 添加 BinanceSpot API 配置
└── 添加 BybitSpot API 配置
```

### 代码行数估算
```
binance/spot_client.go        ~80 行
binance/spot_account_client.go ~100 行
bybit/spot_client.go          ~80 行
bybit/spot_account_client.go   ~100 行
trade_type_manager.go          ~40 行

修改现有文件 ~60 行
─────────────────────────
总计新增: ~460 行
```

---

## 实现步骤

1. **创建 Spot 客户端** (参考 Futures 客户端)
2. **创建 Spot 账户客户端** (参考 Futures 账户客户端)
3. **更新配置** (添加 spot API section)
4. **修改 main.go** (分离初始化逻辑)
5. **修改 Monitor Service** (支持 trade type 选择)
6. **测试** (期货/现货分别测试)

---

## 使用示例

```go
// main.go 中
clients := initializeAPIClients(cfg)

// monitor service 中
svc := monitor.NewService(monitor.ServiceDeps{
    // ... existing fields ...
    FuturesOrderManager: clients.FuturesOrderManager,
    FuturesAccountManager: clients.FuturesAccountManager,
    SpotOrderManager: clients.SpotOrderManager,
    SpotAccountManager: clients.SpotAccountManager,
})

// 在 service.Run() 中动态选择
if symbol == "BTC" {
    s.tradeType = "futures"  // BTC 用期货
} else if symbol == "DOGE" {
    s.tradeType = "spot"     // DOGE 用现货
}
```

---

## 优势

✅ **清晰的架构** - 期货和现货逻辑完全分离  
✅ **易于维护** - 每种交易类型的客户端独立  
✅ **易于扩展** - 添加新交易所只需 4 个文件  
✅ **灵活配置** - 可以独立启用/禁用期货或现货  
✅ **向后兼容** - 现有期货逻辑不需要改变

---

## 建议方案

**如果只是想快速支持 Spot，推荐使用 "通用适配器" 方案：**
- 创建 SpotOrderClient 和 SpotAccountClient
- 使用 TradeTypeManager 包装
- 改动最少，风险最低

**如果想最大程度的灵活性，推荐完整方案：**
- 两套独立的 OrderManager 和 AccountManager
- 支持同时运行期货和现货策略
- 架构最清晰
