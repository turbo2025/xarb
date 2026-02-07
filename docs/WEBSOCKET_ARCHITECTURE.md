# WebSocket 管理器架构设计

## 概述
将 WebSocket 连接管理与 HTTP API 客户端分离，实现关注点清晰分离和灵活扩展。

## 架构组件

### 1. ExchangeClientRegistry（HTTP API 客户端）
**位置**: `internal/infrastructure/factory/exchange_registry.go`

**职责**:
- 管理各交易所的 HTTP API 客户端（Order, Position, Account）
- 支持 Spot 和 Futures 市场
- 强类型结构体组织

**支持的交易所**:
- Binance (Spot + Perpetual)
- Bybit (Spot + Perpetual)
- OKX (Spot + Perpetual)

### 2. WebSocketManager（WebSocket 连接管理）
**位置**: `internal/infrastructure/websocket/manager.go`

**职责**:
- 统一管理各交易所的 WebSocket 连接（流式数据）
- 当前支持：价格源（PriceFeed）
- 为未来扩展预留：订单簿（OrderBook）等

**关键特性**:
- 配置驱动：根据 `config.toml` 的具体 URL 动态初始化连接
- 自动注册：各交易所 `register.go` 在 init() 时自动注册工厂函数
- 支持 Spot 和 Futures：为每种交易所/市场类型配置单独的 WebSocket 连接

**接口示例**:
```go
wsManager := websocket.NewWebSocketManager()
if err := wsManager.Initialize(cfg); err != nil {
    log.Fatal(err)
}

// 获取 Binance Spot 的价格源
binanceSpot := wsManager.BinanceSpot()
if binanceSpot != nil && binanceSpot.PriceFeed != nil {
    feed := binanceSpot.PriceFeed
}
```

### 3. 分布式工厂注册系统

#### A. PriceFeed 注册表
**位置**: `internal/infrastructure/pricefeed/registry.go`

**工作流程**:
1. 各交易所 `register.go` 中定义 init() 函数
2. init() 里调用 `pricefeed.Register(exchangeName, factoryFn)`
3. 当导入交易所包时，自动注册工厂函数

**示例** (`internal/infrastructure/exchange/binance/register.go`):
```go
func init() {
    pricefeed.Register("binance", func(wsURL string) port.PriceFeed {
        return NewPerpetualTickerFeed(wsURL)
    })
}
```

#### B. OrderBook 注册表（为未来预留）
**位置**: `internal/infrastructure/orderbook/registry.go`

**设计目的**:
- 后续添加订单簿数据流时，无需修改现有代码
- 各交易所可在 `order_book_register.go` 中自注册
- 与 PriceFeed 并行初始化

## 数据流

```
config.toml (启用的交易所 + WebSocket URLs)
          ↓
WebSocketManager.Initialize()
          ↓
    ┌─────┴─────┬─────────┬──────────┐
    ↓           ↓         ↓          ↓
 Binance      Bybit      OKX      Bitget
    ↓           ↓         ↓          ↓
register.go (each exchange)
    ↓           ↓         ↓          ↓
pricefeed.Register() + orderbook.Register()
    ↓           ↓         ↓          ↓
    ↑───────────┴─────────┴──────────↑
          ↓
pricefeed.Get() + orderbook.Get()
          ↓
WebSocketManager.registerSpotWebSocket()
WebSocketManager.registerPerpetualWebSocket()
          ↓
ServiceContext.priceFeeds (兼容现有代码)
```

## 与 ServiceContext 的集成

**位置**: `internal/infrastructure/svc/service_context.go`

**初始化流程**:
```go
// 1. 创建 WebSocketManager
wsManager := websocket.NewWebSocketManager()

// 2. 初始化 WebSocket 连接
if err := wsManager.Initialize(cfg); err != nil {
    return err
}
sc.wsManager = wsManager

// 3. 提取 PriceFeed 列表（保持兼容性）
feeds := extractPriceFeedsFromWSManager(wsManager)
sc.priceFeeds = feeds
```

**访问方式**:
```go
// 方式1：获取 PriceFeed 列表（现有代码兼容）
feeds := sc.GetPriceFeeds()

// 方式2：获取完整的 WebSocket 管理器（访问所有连接状态和数据）
wsManager := sc.GetWebSocketManager()
binanceSpot := wsManager.BinanceSpot()
if binanceSpot != nil {
    // 可以访问 PriceFeed、OrderBook 等
}
```

## 扩展指南

### 添加新的 WebSocket 数据源（如 OrderBook）

1. **扩展 OrderBook 接口**（`internal/infrastructure/orderbook/registry.go`）:
   ```go
   type OrderBook interface {
       GetBids() []Order
       GetAsks() []Order
   }
   ```

2. **扩展 WebSocketClients**（`internal/infrastructure/websocket/manager.go`）:
   ```go
   type WebSocketClients struct {
       PriceFeed port.PriceFeed
       OrderBook port.OrderBook  // 新增
   }
   ```

3. **各交易所自注册**（`internal/infrastructure/exchange/*/order_book_register.go`）:
   ```go
   func init() {
       orderbook.Register("binance", NewOrderBookFeed)
   }
   ```

4. **WebSocketManager 中添加初始化逻辑**:
   ```go
   // 在 Initialize() 中添加 OrderBook 初始化
   orderBookFn, ok := orderbook.Get(exchangeName)
   if ok {
       wsClients.OrderBook = orderBookFn(cfg.WsURL)
   }
   ```

### 后续可扩展性

当需要添加新的流式数据源时（e.g., Ticker、Funding Rate Stream）：

1. 创建新的 registry 包（`internal/infrastructure/xxx_feed/registry.go`）
2. 各交易所在相应的 register.go 中注册
3. WebSocketManager 内部初始化并分配给 WebSocketClients
4. 无需修改 factory 或其他已有组件

## 配置示例

```toml
[exchanges.binance]
enabled = true
spot_ws_url = "wss://stream.binance.com:9443/ws"
perpetual_ws_url = "wss://fstream.binance.com/ws"

[exchanges.bybit]
enabled = true
spot_ws_url = "wss://stream.bybit.com/v5/public/spot"
perpetual_ws_url = "wss://stream.bybit.com/v5/public/linear"
```

如果某个交易所不需要 Spot 连接，只需将 `spot_ws_url` 留空：
```toml
[exchanges.okx]
spot_ws_url = ""           # 禁用 Spot WebSocket
perpetual_ws_url = "wss://..." # 仅启用 Futures
```

## 总结

✅ **关注点分离**：HTTP 和 WebSocket 管理独立  
✅ **自动注册**：各交易所包独立管理自己的工厂  
✅ **无硬编码**：新增交易所无需修改工厂代码  
✅ **灵活扩展**：为未来数据源预留接口  
✅ **向后兼容**：现有 `GetPriceFeeds()` 调用继续工作  
