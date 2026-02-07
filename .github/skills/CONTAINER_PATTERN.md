# 依赖注入容器模式实现指南

本文档说明如何使用 Container 模式改进依赖管理。

## 概述

新的 Container 模式将所有依赖初始化和生命周期管理集中在一处，使代码更加清晰、可测试和可维护。

## 架构对比

### 之前：直接在 main 中初始化
```go
// ❌ main.go - 混乱且难以维护
func main() {
    // 141 行初始化代码...
    var repoList []monitor.RepositoryCloser
    
    if cfg.Storage.Redis.Enabled {
        rdb := redis.NewClient(...)
        repos = append(repos, redisrepo.New(...))
    }
    
    if cfg.Storage.SQLite.Enabled {
        r, err := sqliterepo.New(...)
        repos = append(repos, r)
        repoList = append(repoList, r)
    }
    
    defer func() {
        for _, c := range repoList {
            _ = c.Close()
        }
    }()
}
```

### 之后：使用 Container 模式
```go
// ✅ main.go - 清晰简洁
func main() {
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatal().Err(err).Msg("load config failed")
    }
    
    // 一行初始化所有依赖
    cont, err := container.New(cfg)
    if err != nil {
        log.Fatal().Err(err).Msg("container initialization failed")
    }
    defer cont.Close()
    
    // 使用依赖...
    redisRepo := cont.RedisRepo()
}
```

## Container 的职责

### 1. 配置注入
```go
// 访问配置
cfg := cont.Config()
```

### 2. 存储层初始化
- **Redis**：连接到 Redis 服务器并创建仓储
- **SQLite**：初始化数据库并执行迁移
- **Postgres**：连接数据库（可选）

### 3. 资源验证
```go
// Redis ping 测试
if err := rdb.Ping(ctx).Err(); err != nil {
    return fmt.Errorf("redis ping failed: %w", err)
}
```

### 4. 生命周期管理
```go
// 按后进先出顺序关闭所有资源
defer cont.Close()
```

## 使用示例

### 基本使用
```go
func main() {
    cfg, _ := config.Load("configs/config.toml")
    cont, _ := container.New(cfg)
    defer cont.Close()
    
    // 获取配置
    symbols := cont.Config().Symbols.List
    
    // 获取 Redis 仓储
    redisRepo := cont.RedisRepo()
    
    // 获取 SQLite 仓储
    sqliteRepo := cont.SQLiteRepo()
}
```

### 在服务中使用
```go
type MyService struct {
    redisRepo *redisrepo.Repo
    sqliteRepo *sqliterepo.Repo
}

func NewMyService(cont *container.Container) *MyService {
    return &MyService{
        redisRepo: cont.RedisRepo(),
        sqliteRepo: cont.SQLiteRepo(),
    }
}
```

### 在测试中模拟
```go
func TestMyService(t *testing.T) {
    cfg := &config.Config{...}
    cfg.Storage.Redis.Enabled = false
    cfg.Storage.SQLite.Enabled = true
    
    cont, _ := container.New(cfg)
    defer cont.Close()
    
    svc := NewMyService(cont)
    // 测试...
}
```

## 主要优势

| 方面 | 改进 |
|------|------|
| **代码清晰度** | main.go 从 141 行减少到 ~50 行 |
| **初始化顺序** | 集中管理，易于理解 |
| **错误处理** | 统一的初始化失败处理 |
| **资源清理** | 自动按正确顺序关闭 |
| **测试性** | 易于创建测试用的 Container |
| **扩展性** | 添加新依赖无需修改 main.go |
| **配置访问** | 全局访问配置：`cont.Config()` |

## 扩展 Container

### 添加新的依赖（例如：Postgres）
```go
// 在 Container 中添加字段
type Container struct {
    postgresRepo *pgrepo.Repo
    // ...
}

// 实现初始化方法
func (c *Container) initPostgres() error {
    repo, err := pgrepo.New(c.cfg.Storage.Postgres.DSN)
    if err != nil {
        return err
    }
    c.postgresRepo = repo
    c.closerChain = append(c.closerChain, func() error {
        return repo.Close()
    })
    return nil
}

// 添加访问方法
func (c *Container) PostgresRepo() *pgrepo.Repo {
    return c.postgresRepo
}
```

### 添加新的组件（例如：Logger）
```go
type Container struct {
    logger zerolog.Logger
    // ...
}

func (c *Container) Logger() zerolog.Logger {
    return c.logger
}
```

## 错误处理流程

```
New(cfg)
    ↓
initStorage()
    ├─ initRedis()    ← 可能失败
    ├─ initSQLite()   ← 可能失败
    └─ initPostgres() ← 可能失败
    ↓
如果任何步骤失败 → Close() 清理已初始化的资源 → 返回错误
```

## Context 中的配置访问

虽然目前还没有在 context 中存储配置，但如果需要可以这样扩展：

```go
// 在某个处理函数中
type configKey struct{}

ctx = context.WithValue(ctx, configKey{}, cont.Config())

// 在其他地方检索
cfg := ctx.Value(configKey{}).(*config.Config)
```

但**推荐的做法**是通过 Container 的 getter 方法访问：
```go
cfg := cont.Config()
```

## 最佳实践

✅ **要做的：**
- 在 main 中创建一个 Container 实例
- 使用 `defer cont.Close()` 确保资源释放
- 通过 Container 的 getter 方法访问依赖
- 在测试中创建专用的 Container 配置

❌ **不要做的：**
- 在不同的地方创建多个 Container 实例
- 绕过 Container 直接初始化依赖
- 忘记调用 `Close()`
- 在 goroutine 中创建 Container（不安全）

## 参考

- [Go 代码规范](../.github/skills/GO_CONVENTIONS.md#依赖注入)
- [current main.go implementation](../cmd/xarb/main.go)
- [Container implementation](container.go)
