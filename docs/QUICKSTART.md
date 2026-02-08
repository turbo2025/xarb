# 快速参考指南

## 编译和运行

```bash
# 编译
go build -o xarb ./cmd/xarb

# 或使用 Makefile
make build

# 运行
./xarb --config configs/config.toml

# 开发模式（监听文件变化）
make dev
```

## 常用命令

```bash
# 测试
make test          # 运行所有测试
make test-arbitrage  # 只运行套利系统测试
make test-cover     # 生成覆盖率报告

# 代码质量
make fmt           # 格式化代码
make lint          # 代码检查
make tidy          # 整理依赖

# 清理
make clean         # 删除编译产物
make help          # 显示所有命令
```

## 配置说明

编辑 `configs/config.toml`:

```toml
[app]
# 每 N 分钟打印一次统计
print_every_min = 1

[symbols]
# 监控的交易对
list = ["BTCUSDT", "ETHUSDT", "BNBUSDT"]

[exchange.binance]
enabled = true
ws_url = "wss://fstream.binance.com"
balance = 500  # USDT

[exchange.bybit]
enabled = true
ws_url = "wss://stream.bybit.com/v5/public/linear"
balance = 500

[exchange.okx]
enabled = false
ws_url = "wss://ws.okx.com:8443/ws/v5/public"
balance = 500

[exchange.bitget]
enabled = false
ws_url = "wss://ws.bitget.com/spot/v1/public"
balance = 500

[storage.enabled]
true

[storage.sqlite]
enabled = true
path = "data/xarb.db"

[storage.redis]
enabled = false
addr = "localhost:6379"
password = ""
db = 0
prefix = "xarb:"
ttl_seconds = 3600
signal_stream = "signals"
signal_channel = "alerts"

[arbitrage]
# 最小可交易的价差百分比（%）
min_spread = 0.01
# 价差变化阈值，用于触发告警
delta_threshold = 0.5
```

## 核心 API

### 计算器

```go
calc := service.NewArbitrageCalculator(0.0002)

// 计算价差
spread := calc.CalculateSpread(binancePrice, bybitPrice, 0.0002)

// 计算资金费
funding := calc.CalculateFunding(binancePrice, bybitPrice, 24)
```

### 服务

```go
svc := service.NewArbitrageService(repo, calc, 0.01, 0.0002)

// 扫描价差机会
svc.ScanSpreadOpportunities(ctx, price1, price2)

// 扫描资金费机会
svc.ScanFundingOpportunities(ctx, price1, price2, 24)

// 开仓
svc.OpenPosition(ctx, "BTCUSDT", "binance", "bybit", 0.1, 43000, 42900)

// 获取盈亏
pnl, _ := svc.GetPositionPnL(ctx, posID)
```

### 数据库

```go
// 初始化
repo, _ := sqlite.New("data/xarb.db")
defer repo.Close()

arbRepo := sqlite.NewArbitrageRepo(repo.GetDB())

// 保存机会
arbRepo.SaveSpreadOpportunity(ctx, spread)
arbRepo.SaveFundingOpportunity(ctx, funding)

// 查询机会
spread, _ := arbRepo.GetLatestSpreadBySymbol(ctx, "BTCUSDT")
funding, _ := arbRepo.GetLatestFundingBySymbol(ctx, "ETHUSDT")

// 管理持仓
arbRepo.CreatePosition(ctx, position)
arbRepo.UpdatePosition(ctx, position)
positions, _ := arbRepo.ListOpenPositions(ctx)
```

## 日志

系统使用 `rs/zerolog` 记录，输出 JSON 格式。

在代码中：
```go
log.Info().
    Str("symbol", "BTCUSDT").
    Float64("spread", 0.23).
    Msg("spread opportunity detected")
```

输出：
```json
{"level":"info","symbol":"BTCUSDT","spread":0.23,"time":"2026-02-08T12:00:00+08:00","message":"spread opportunity detected"}
```

## 数据库查询示例

### 查看最新价差机会

```sql
SELECT * FROM spread_opportunities 
WHERE symbol = 'BTCUSDT' 
ORDER BY created_at DESC 
LIMIT 10;
```

### 查看开仓持仓

```sql
SELECT * FROM arbitrage_positions 
WHERE status = 'open'
ORDER BY open_time DESC;
```

### 查看24小时利润

```sql
SELECT 
    symbol,
    SUM(realized_pnl) as daily_pnl,
    COUNT(*) as closed_positions
FROM arbitrage_positions
WHERE close_time > datetime('now', '-24 hours')
GROUP BY symbol;
```

### 查看资金费趋势

```sql
SELECT 
    symbol,
    long_exchange,
    short_exchange,
    AVG(funding_diff) as avg_diff,
    MAX(funding_diff) as max_diff,
    MIN(funding_diff) as min_diff
FROM funding_opportunities
WHERE created_at > datetime('now', '-7 days')
GROUP BY symbol, long_exchange, short_exchange;
```

## 常见问题

### Q: 如何增加新的交易所？
A: 在 `internal/infrastructure/exchange/` 下创建新目录，实现 `PriceFeed` 接口，然后在 `main.go` 的 `initializeFeeds()` 中添加初始化代码。

### Q: 如何修改手续费率？
A: 在 `NewArbitrageCalculator(makerFeePercentage)` 中传入正确的费率值（如 0.0002 表示 0.02%）。

### Q: 如何切换数据库？
A: 修改 `configs/config.toml` 中的 `storage` 部分，启用 redis 或 postgres，修改对应的配置。

### Q: 如何自定义最小价差？
A: 在配置中修改 `arbitrage.min_spread` 或在代码中调用服务时传入参数。

### Q: 如何调试？
A: 
```bash
# 启用 debug 日志
RUST_LOG=debug ./xarb

# 或者在代码中设置日志级别
// logger.Setup() 中可配置
```

## 部署建议

### 开发环境
```bash
make build
./xarb --config configs/config.toml
```

### 生产环境
```bash
# 使用 SQLite（轻量）
./xarb --config configs/config.production.toml

# 或使用 PostgreSQL（生产级）
# 配置 PostgreSQL 连接后运行
./xarb --config configs/config.postgres.toml
```

### Docker 部署
```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o xarb ./cmd/xarb

FROM alpine:latest
COPY --from=builder /app/xarb .
COPY configs/ configs/
CMD ["./xarb", "--config", "configs/config.toml"]
```

## 性能优化建议

1. **增加 WebSocket 连接数**: 在配置中启用更多交易所
2. **启用 Redis 缓存**: 减少 SQLite I/O
3. **批量插入**: 积累多个机会后批量保存
4. **索引优化**: 根据查询模式在需要的列上添加索引

## 监控和告警

推荐集成：
- Prometheus 导出指标
- Grafana 可视化
- AlertManager 告警

关键指标：
- 价差分布
- 资金费变化
- 持仓数量和利润
- WebSocket 连接状态

---

更多信息见 `ARBITRAGE.md` 和 `PROJECT_SUMMARY.md`
