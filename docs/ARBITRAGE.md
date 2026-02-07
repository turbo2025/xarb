# XARB - 跨交易所永续合约套利系统

完整的跨交易所套利系统，支持 Binance 和 Bybit 的三种核心套利策略。

## 系统架构

### 核心组件

1. **Exchange WebSocket Feeds** - 实时价格流
   - Binance Futures Mini Ticker
   - Bybit Linear Ticker
   - OKX Public Ticker
   - Bitget Market Ticker

2. **Storage Layer** - 数据持久化
   - SQLite - 本地存储（推荐开发/轻量部署）
   - Redis - 缓存和实时流（可选）
   - PostgreSQL - 生产级存储（可选）

3. **Domain Models** - 核心概念
   - `FuturesPrice` - 永续合约价格，包含资金费率
   - `SpreadArbitrage` - 价差套利机会
   - `FundingArbitrage` - 资金费率套利
   - `ArbitragePosition` - 持仓管理

4. **Services** - 业务逻辑
   - `ArbitrageCalculator` - 套利计算引擎
   - `ArbitrageService` - 机会扫描和持仓管理

## 三种套利策略

### 1. 价差收敛 (Spread Arbitrage)

在两个交易所同时做多和做空，利用价格差异套利。

**原理:**
- 在低价交易所做多（买入）
- 在高价交易所做空（卖出）
- 等待价差收敛时平仓

**计算示例:**
```
Binance BTCUSDT: 43,000 USDT
Bybit BTCUSDT:   42,900 USDT （便宜100 USDT）

价差百分比 = (43000 - 42900) / 42900 * 100% = 0.233%

手续费成本 (maker 0.02%) = (43000*0.0002 + 42900*0.0002) / 42900 * 100% = 0.040%

预期利润 = 0.233% - 0.040% = 0.193%
```

### 2. 资金费率差异 (Funding Rate Arbitrage)

利用两个交易所资金费率差异获利。

**原理:**
- 在资金费高的交易所做空
- 在资金费低的交易所做多
- 持仓赚取资金费差异

**计算示例:**
```
Binance ETHUSDT: 资金费 0.03% 
Bybit ETHUSDT:   资金费 0.01% （便宜0.02%）

资金费差 = 0.03% - 0.01% = 0.02%
8小时结算周期，每周期赚取0.02%
24小时 = 3个周期，预期回报 = 0.02% * 3 = 0.06%
```

### 3. 做市返佣 (Maker Rebate)

利用交易所的 maker 返佣降低成本或增加收益。

## 快速开始

### 编译

```bash
make build
```

### 配置

编辑 `configs/config.toml`:

```toml
[app]
print_every_min = 1

[symbols]
list = ["BTCUSDT", "ETHUSDT"]

[exchange.binance]
enabled = true
ws_url = "wss://fstream.binance.com"
balance = 500  # 该交易所分配500 USDT

[exchange.bybit]
enabled = true
ws_url = "wss://stream.bybit.com/v5/public/linear"
balance = 500

[storage.sqlite]
enabled = true
path = "data/xarb.db"

[arbitrage]
min_spread = 0.01  # 最小0.01%利润才交易
delta_threshold = 0.5
```

### 运行

```bash
# 开发/测试模式
./xarb --config configs/config.toml

# 或使用make命令
make run
```

## API 使用示例

### 创建套利计算器

```go
calc := service.NewArbitrageCalculator(0.0002)  // 0.02% maker 费率

// 计算价差机会
spread := calc.CalculateSpread(binancePrice, bybitPrice, 0.0002)

// 计算资金费机会
funding := calc.CalculateFunding(binancePrice, bybitPrice, 24)
```

### 管理持仓

```go
svc := service.NewArbitrageService(repo, calc, 0.01, 0.0002)

// 扫描价差机会
err := svc.ScanSpreadOpportunities(ctx, price1, price2)

// 扫描资金费机会
err := svc.ScanFundingOpportunities(ctx, price1, price2, 24)

// 开仓
err := svc.OpenPosition(ctx, "BTCUSDT", "binance", "bybit", 0.1, 43000, 42900)

// 获取盈亏
pnl, err := svc.GetPositionPnL(ctx, "pos_001")
```

## 数据库表结构

### 价差机会表
```sql
CREATE TABLE spread_opportunities (
  id INTEGER PRIMARY KEY,
  symbol TEXT,
  long_exchange TEXT,
  short_exchange TEXT,
  long_price REAL,
  short_price REAL,
  spread REAL,
  spread_abs REAL,
  profit_percent REAL,
  ts_ms INTEGER,
  created_at INTEGER
);
```

### 资金费机会表
```sql
CREATE TABLE funding_opportunities (
  id INTEGER PRIMARY KEY,
  symbol TEXT,
  long_exchange TEXT,
  short_exchange TEXT,
  long_funding REAL,
  short_funding REAL,
  funding_diff REAL,
  holding_hours INTEGER,
  expected_return REAL,
  ts_ms INTEGER,
  created_at INTEGER
);
```

### 持仓表
```sql
CREATE TABLE arbitrage_positions (
  id TEXT PRIMARY KEY,
  symbol TEXT,
  long_exchange TEXT,
  short_exchange TEXT,
  quantity REAL,
  long_entry_price REAL,
  short_entry_price REAL,
  entry_spread REAL,
  status TEXT,  -- open, closing, closed
  open_time INTEGER,
  close_time INTEGER,
  realized_pnl REAL,
  created_at INTEGER,
  updated_at INTEGER
);
```

## 测试

```bash
# 运行所有测试
make test

# 运行套利系统测试
go test ./internal/application/service -run Arbitrage -v

# 运行数据库集成测试
go test ./internal/infrastructure/storage/sqlite -run Arbitrage -v
```

## 性能指标

- **价差扫描**: 毫秒级
- **资金费查询**: 秒级（受交易所API限制，约10秒）
- **数据库吞吐**: SQLite 支持 >10K ops/sec
- **WebSocket延迟**: <100ms（交易所相关）
- **账户信息查询**: 秒级（HTTP REST API）

## 风险管理建议

1. **头寸大小**: 每个交易所 500 USDT
2. **滑点防护**: 实际执行可能遇到 0.1-0.5% 滑点
3. **资金费周期**: Binance 每 8 小时，Bybit 每 1 小时
4. **对手风险**: 只使用顶级交易所
5. **价差门槛**: 至少 0.2% 才值得交易（覆盖手续费和滑点）

## 文件结构

```
xarb/
├── cmd/xarb/              # 可执行程序入口
├── configs/               # 配置文件
├── data/                  # 数据存储
├── internal/
│   ├── application/       # 应用层
│   │   ├── port/          # 接口定义
│   │   ├── service/       # 业务逻辑协调
│   │   ├── container/     # DI容器
│   │   └── usecase/       # 用例实现
│   ├── domain/            # 领域层
│   │   ├── model/         # 领域模型
│   │   └── service/       # 核心业务逻辑
│   ├── infrastructure/    # 基础设施层
│   │   ├── exchange/      # 交易所集成
│   │   ├── storage/       # 数据存储
│   │   ├── config/        # 配置管理
│   │   ├── factory/       # 工厂帮手
│   │   └── container/     # 容器实现
│   └── interfaces/        # 接口层
│       ├── console/       # CLI输出
│       └── http/          # REST API
├── go.mod                 # Go模块
├── go.sum                 # 依赖
├── Makefile               # 编译脚本
└── docs/                  # 文档
```

## 许可证

MIT
