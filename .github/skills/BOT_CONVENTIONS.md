# 机器人（Bot）规范

本文档定义了加密货币套利机器人的开发、部署和运维规范。

## 目录

- [机器人类型](#机器人类型)
- [开发规范](#开发规范)
- [配置管理](#配置管理)
- [运维规范](#运维规范)
- [监控告警](#监控告警)
- [安全规范](#安全规范)

## 机器人类型

本项目包含以下类型的机器人：

### 1. 监控机器人（Monitor Bot）
- **功能**：监听交易所价格变化，检测套利机会
- **职责**：
  - 实时获取多个交易所的交易对价格
  - 计算价差和收益率
  - 格式化和输出监控数据
  - 存储历史数据用于分析

- **关键组件**：
  - `EventBus`：发布价格更新事件
  - `PriceFeed`：接收价格数据
  - `Repository`：存储监控数据
  - `Sink`：输出监控结果

## 开发规范

### 1. 机器人架构

所有机器人应遵循分层架构：

```
┌─────────────────────────────────────┐
│         User Interface              │ (HTTP Server, Console)
├─────────────────────────────────────┤
│       Application/Usecase Layer     │ (业务流程编排)
├─────────────────────────────────────┤
│        Domain/Business Logic        │ (核心算法、规则)
├─────────────────────────────────────┤
│      Infrastructure Layer           │ (数据库、API、消息队列)
└─────────────────────────────────────┘
```

### 2. 依赖管理

```
                    ┌──────────────┐
                    │  UseCase     │
                    └──────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
    ┌────────┐      ┌──────────┐      ┌──────────┐
    │ Logger │      │Repository│      │EventBus  │
    └────────┘      └──────────┘      └──────────┘
```

所有依赖通过构造函数注入，禁止使用全局变量。

### 3. 生命周期管理

所有机器人应实现以下接口：

```go
type Bot interface {
    Start(ctx context.Context) error
    Stop() error
    Health() error
}
```

### 4. 配置管理

机器人配置应包括：

```toml
[bot]
name = "monitor"
enabled = true
workers = 4
log_level = "info"

[exchanges]
[exchanges.binance]
enabled = true
ws_url = "wss://stream.binance.com:9443/ws"
symbols = ["BTCUSDT", "ETHUSDT"]
timeout = 10

[exchanges.bybit]
enabled = true
ws_url = "wss://stream.bybit.com/v5/public/spot"
symbols = ["BTCUSDT", "ETHUSDT"]
timeout = 10

[storage]
type = "postgres"  # 或 redis, sqlite
host = "localhost"
port = 5432
database = "xarb"

[alert]
enabled = true
min_spread = 0.5  # 最小套利空间百分比
notification_urls = ["http://webhook.example.com/alert"]
```

## 配置管理

### 1. 环境变量

使用环境变量覆盖敏感信息：

```bash
export DB_PASSWORD=xxx
export API_KEY=xxx
export API_SECRET=xxx
```

### 2. 配置加载顺序

1. 读取 `configs/config.toml`
2. 用环境变量覆盖配置
3. 验证必要配置项

### 3. 配置验证

启动时必须验证：

```go
func (c *Config) Validate() error {
    if c.Bot.Name == "" {
        return errors.New("bot name is required")
    }
    if len(c.Exchanges) == 0 {
        return errors.New("at least one exchange must be configured")
    }
    // 更多验证...
    return nil
}
```

## 运维规范

### 1. 启动流程

```
1. 加载配置和验证
2. 初始化日志系统
3. 连接数据库
4. 初始化 WebSocket 连接
5. 启动业务逻辑
6. 暴露 HTTP 端点用于健康检查
```

### 2. 优雅关闭

```go
// 收到关闭信号时：
1. 停止接收新请求
2. 等待进行中的操作完成（带超时）
3. 关闭数据库连接
4. 关闭 WebSocket 连接
5. 记录关闭日志
```

### 3. 日志规范

不同日志级别的使用：

- **ERROR**：机器人无法继续运行的错误
  ```go
  logger.Error("failed to connect to exchange", "exchange", "binance", "error", err)
  ```

- **WARN**：降级或临时问题
  ```go
  logger.Warn("price feed latency high", "latency_ms", 500)
  ```

- **INFO**：重要业务事件
  ```go
  logger.Info("arbitrage opportunity detected", "spread", 0.5)
  ```

- **DEBUG**：调试信息（仅在开发时启用）
  ```go
  logger.Debug("processing message", "message", msg)
  ```

### 4. 数据持久化

- 使用事务确保数据一致性
- 实现重试机制处理临时故障
- 定期备份重要数据

## 监控告警

### 1. 关键指标

每个机器人应暴露以下指标：

```
# 价格相关
- current_price{exchange, symbol}
- price_update_lag{exchange}  (ms)

# 套利相关
- spread_percentage{pair}
- arbitrage_opportunities_count
- executed_arbitrage_profit{symbol}

# 系统相关
- message_queue_depth
- database_connection_pool_idle
- websocket_reconnect_count
- error_rate
```

### 2. 告警规则

```
# 交易所连接告警
- 如果5分钟内无价格更新，触发告警

# 数据异常告警
- 价格跳变超过 10% 触发告警
- 连续错误超过 5 次触发告警

# 性能告警
- 消息处理延迟超过 1 秒触发告警
- 数据库连接失败触发告警
```

### 3. 健康检查

```go
type HealthStatus struct {
    Status   string            `json:"status"`      // "healthy", "degraded", "unhealthy"
    Checks   map[string]bool   `json:"checks"`
    LastCheck time.Time        `json:"last_check"`
}

// 健康检查应验证：
// - 数据库连接
// - WebSocket 连接
// - 消息处理能力
// - 存储可用性
```

HTTP 端点：
```
GET /health       # 简化健康检查
GET /health/full  # 详细健康检查
```

## 安全规范

### 1. API 密钥管理

- ✅ 使用环境变量存储敏感信息
- ✅ 使用密钥管理服务（如 HashiCorp Vault）
- ✅ 定期轮换密钥
- ✅ 限制密钥权限到最小必需

不要做：
- ❌ 在代码中硬编码密钥
- ❌ 在日志中打印完整密钥
- ❌ 在版本控制中提交密钥

### 2. 数据传输安全

- 使用 HTTPS/WSS 进行网络通信
- 验证 SSL 证书
- 使用加密存储敏感数据

### 3. 访问控制

- 限制机器人的 API 权限到必要的操作
  - 仅读权限：获取价格、余额
  - 非写权限：禁用提现、修改密钥
- 实施 IP 白名单（如适用）
- 记录所有 API 调用用于审计

### 4. 更新安全

- 定期更新依赖包
- 使用 `go mod tidy` 和 `go mod verify` 验证依赖
- 监控已知漏洞：
  ```bash
  go install github.com/golang/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

## 最佳实践

### 1. 优雅降级

当交易所不可用时：

```go
// 使用缓存的价格数据
// 发送告警通知
// 继续尝试重新连接
```

### 2. 速率限制

遵守交易所的 API 限制：

```go
const (
    MaxRequestsPerSecond = 10
    MaxConcurrentRequests = 3
)
```

### 3. 错误恢复

```go
// 指数退避重试
attempts := 0
maxAttempts := 5
backoff := 1 * time.Second

for attempts < maxAttempts {
    err := operation()
    if err == nil {
        break
    }
    attempts++
    time.Sleep(backoff)
    backoff *= 2
}
```

### 4. 资源清理

```go
// 使用 defer 确保资源被释放
func (b *Bot) Process(ctx context.Context) error {
    conn := acquireConnection()
    defer conn.Close()
    
    // 处理逻辑
}
```

## 发布检查清单

机器人发布前检查：

- [ ] 所有单元测试通过
- [ ] 代码覆盖率 >= 80%
- [ ] 配置已验证
- [ ] 日志级别设置正确（DEBUG → INFO）
- [ ] 敏感信息已移至环境变量
- [ ] 依赖已更新且安全
- [ ] 文档已更新
- [ ] 监控告警已配置
- [ ] 健康检查端点已就绪
- [ ] 灾备计划已准备

## 参考资源

- [Go 代码规范](GO_CONVENTIONS.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- 交易所 API 文档
