# Storage 初始化内聚化改进

## 变化总结

✅ **Redis 和 SQLite 初始化从 Container 移入 ServiceContext**

### Before (旧设计)
```
main.go
  └─ svc.New(ctx, cfg)
      └─ container.New(cfg)
          ├─ initRedis()
          ├─ initSQLite()
          └─ return Container
      
问题:
- 初始化逻辑分散到两个地方
- 看不清楚初始化的全貌
- Container 是一个额外的中间层
```

### After (新设计)
```
main.go
  └─ svc.New(ctx, cfg)
      └─ initializeComponents()
          ├─ [Step 0] initializeStorage()
          │   ├─ initRedis()     ← 直接在 ServiceContext 中
          │   └─ initSQLite()    ← 直接在 ServiceContext 中
          │
          ├─ [Step 1] ArbitrageCalculator
          ├─ [Step 2] SymbolMapper
          ├─ [Step 3] APIClients
          ├─ [Step 4] OrderManager/AccountManager
          └─ [Step 5] PriceFeeds

优势:
✅ 初始化流程清晰有序
✅ 一个地方可看到全部初始化
✅ 存储层优先级明确 (Step 0)
✅ 错误处理一致
✅ 资源清理统一 (closerChain)
```

## 初始化顺序的逻辑

```
依赖关系树:

存储层 (Redis/SQLite)  ← 最基础的依赖 (Step 0)
  │
  ├─ ArbitrageCalculator  ← 基础组件，无依赖 (Step 1)
  │
  ├─ SymbolMapper  ← Domain 层，无外部依赖 (Step 2)
  │
  ├─ APIClients  ← 依赖配置，独立初始化 (Step 3)
  │
  ├─ OrderManager  ← 依赖 APIClients (Step 4)
  │
  └─ PriceFeeds  ← 网络连接，最后初始化 (Step 5)
```

## 代码改进

### ServiceContext 结构体

```go
type ServiceContext struct {
	Ctx context.Context
	Config *config.Config
	Sink port.Sink

	// 存储层组件 (新增细节)
	redisClient     *redisclient.Client
	sqliteRepo      *sqliterepo.Repo
	sqliteArbRepo   *sqliterepo.ArbitrageRepo  // 套利仓储
	redisRepo       *redisrepo.Repo

	// 应用业务组件
	priceFeeds            []monitor.PriceFeed
	arbitrageCalculator   *service.ArbitrageCalculator
	symbolMapper          *domainservice.SymbolMapper
	tradeTypeManager      *domainservice.TradeTypeManager
	arbitrageExecutor     *domainservice.ArbitrageExecutor
	futuresOrderManager   *domainservice.OrderManager
	futuresAccountManager *domainservice.AccountManager

	// 资源清理链 (新增)
	closerChain []func() error
}
```

### 初始化函数

```go
func (sc *ServiceContext) initializeComponents() error {
	// 0️⃣ 存储层优先初始化
	if sc.Config.Storage.Enabled {
		if err := sc.initializeStorage(); err != nil {
			return fmt.Errorf("storage initialization failed: %w", err)
		}
	}

	// 1️⃣ 基础组件
	sc.arbitrageCalculator = service.NewArbitrageCalculator(0.0002)

	// 2️⃣ Domain 层
	sc.symbolMapper = domainservice.NewSymbolMapper()
	if err := sc.symbolMapper.LoadDefaultConfig(); err != nil {
		log.Warn().Err(err).Msg("failed to load default symbol mapping")
	}

	// 3️⃣ Infrastructure - 交易所 API
	clients := factory.NewAPIClients(sc.Config)
	sc.tradeTypeManager = clients.TradeTypeManager
	sc.arbitrageExecutor = clients.ArbitrageExecutor

	// 4️⃣ 订单和账户管理
	var err1, err2 error
	sc.futuresOrderManager, err1 = sc.tradeTypeManager.GetOrderManager("futures")
	sc.futuresAccountManager, err2 = sc.tradeTypeManager.GetAccountManager("futures")
	if err1 != nil || err2 != nil {
		log.Warn().Msg("failed to get futures clients, continuing with spot only")
	}

	// 5️⃣ 价格源 (最后)
	feeds := factory.NewPriceFeeds(sc.Config)
	if len(feeds) == 0 {
		return ErrNoFeedsEnabled
	}
	sc.priceFeeds = feeds

	return nil
}

func (sc *ServiceContext) initializeStorage() error {
	if sc.Config.Storage.Redis.Enabled {
		if err := sc.initRedis(); err != nil {
			return fmt.Errorf("redis initialization failed: %w", err)
		}
	}

	if sc.Config.Storage.SQLite.Enabled {
		if err := sc.initSQLite(); err != nil {
			return fmt.Errorf("sqlite initialization failed: %w", err)
		}
	}

	return nil
}

func (sc *ServiceContext) initRedis() error {
	rdb := redisclient.NewClient(&redisclient.Options{
		Addr:     sc.Config.Storage.Redis.Addr,
		Password: sc.Config.Storage.Redis.Password,
		DB:       sc.Config.Storage.Redis.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(sc.Ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	sc.redisClient = rdb
	ttl := time.Duration(sc.Config.Storage.Redis.TTLSeconds) * time.Second

	sc.redisRepo = redisrepo.New(
		rdb,
		sc.Config.Storage.Redis.Prefix,
		ttl,
		sc.Config.Storage.Redis.SignalStream,
		sc.Config.Storage.Redis.SignalChannel,
	)

	// 注册关闭回调 (新增)
	sc.closerChain = append(sc.closerChain, func() error {
		log.Info().Msg("closing redis connection")
		return rdb.Close()
	})

	log.Info().
		Str("addr", sc.Config.Storage.Redis.Addr).
		Int("db", sc.Config.Storage.Redis.DB).
		Msg("✓ Redis initialized")

	return nil
}

func (sc *ServiceContext) initSQLite() error {
	repo, err := sqliterepo.New(sc.Config.Storage.SQLite.Path)
	if err != nil {
		return fmt.Errorf("sqlite repo creation failed: %w", err)
	}

	sc.sqliteRepo = repo
	sc.sqliteArbRepo = sqliterepo.NewArbitrageRepo(repo.GetDB())  // 新增

	// 注册关闭回调 (新增)
	sc.closerChain = append(sc.closerChain, func() error {
		log.Info().Msg("closing sqlite connection")
		return repo.Close()
	})

	log.Info().
		Str("path", sc.Config.Storage.SQLite.Path).
		Msg("✓ SQLite initialized")

	return nil
}

// 资源清理 (改进)
func (sc *ServiceContext) Close() error {
	// 关闭价格源
	if sc.priceFeeds != nil {
		for _, feed := range sc.priceFeeds {
			if closeable, ok := feed.(interface{ Close() error }); ok {
				if err := closeable.Close(); err != nil {
					log.Error().Err(err).Msg("error closing price feed")
				}
			}
		}
	}

	// 按照相反的顺序关闭资源 (LIFO)
	for i := len(sc.closerChain) - 1; i >= 0; i-- {
		if err := sc.closerChain[i](); err != nil {
			log.Error().Err(err).Msg("error closing resource")
		}
	}

	return nil
}

// Getter 方法 (新增)
func (sc *ServiceContext) GetSQLiteRepo() *sqliterepo.Repo {
	return sc.sqliteRepo
}

func (sc *ServiceContext) GetRedisRepo() *redisrepo.Repo {
	return sc.redisRepo
}
```

## 优势分析

### 1. **完全内聚化**
- ✅ 所有初始化在一个地方：ServiceContext
- ✅ 不需要查看 Container 源码就能理解初始化流程
- ✅ 清晰的依赖关系展示

### 2. **错误处理统一**
```go
// 现在所有初始化错误都有明确的来源
if sc.Config.Storage.Enabled {
    if err := sc.initializeStorage(); err != nil {
        return fmt.Errorf("storage initialization failed: %w", err)
    }
}
```

### 3. **资源清理可靠**
```go
// 使用 closerChain 确保所有资源都被清理
// 按照 LIFO 顺序关闭，避免依赖问题
for i := len(sc.closerChain) - 1; i >= 0; i-- {
    if err := sc.closerChain[i](); err != nil {
        log.Error().Err(err).Msg("error closing resource")
    }
}
```

### 4. **初始化顺序透明**
```
Step 0️⃣  Storage (Redis/SQLite)  ← 最基础
         │
Step 1️⃣  ArbitrageCalculator  ← 无依赖
         │
Step 2️⃣  SymbolMapper        ← Domain
         │
Step 3️⃣  APIClients          ← Infrastructure
         │
Step 4️⃣  OrderManager        ← 依赖 APIClients
         │
Step 5️⃣  PriceFeeds          ← 网络，最后
```

### 5. **Getter 方法便利**
```go
// 其他模块可以随时访问已初始化的存储
sqliteRepo := sc.GetSQLiteRepo()
redisRepo := sc.GetRedisRepo()
```

## 代码行数变化

| 模块 | 改变 |
|------|------|
| service_context.go | 159 → 283 行 (+124, 78%) |
| main.go | 不变 (59 行) |
| 总体 | 代码更清晰，初始化更内聚 |

## 与 DDD 的兼容性

✅ **完全兼容**

- **Domain Layer**: 零变化，仍然零依赖
- **Application Layer**: 仍然通过 Port 接口使用存储
- **Infrastructure Layer**: 实现细节仍然隐藏
- **ServiceContext**: 位置依然在 Infrastructure 层

## 推荐做法

### 访问存储的模式

```go
// ❌ 不推荐: 直接访问 StorageContainer
//repo := storageContainer.SQLiteArbitrageRepo()

// ✅ 推荐: 通过 ServiceContext Getter
deps := sc.BuildMonitorServiceDeps()
// ArbitrageRepo 已经包含在 deps 中

// 或者直接
sqliteRepo := sc.GetSQLiteRepo()
```

## 总结

✅ **Storage 初始化完全内聚到 ServiceContext**
✅ **初始化顺序清晰明确 (6 步有序流程)**
✅ **错误处理和资源清理统一**
✅ **与 DDD 分层完全兼容**
✅ **提供清晰的 Getter 方法访问**

这个改进使得整个应用的启动过程更加清晰、可维护、可控。
