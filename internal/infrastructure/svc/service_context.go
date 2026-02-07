package svc

import (
	"context"
	"fmt"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"xarb/internal/application/port"
	"xarb/internal/application/service"
	"xarb/internal/application/usecase/monitor"
	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/factory"
	redisrepo "xarb/internal/infrastructure/storage/redis"
	sqliterepo "xarb/internal/infrastructure/storage/sqlite"
	"xarb/internal/infrastructure/websocket"
	"xarb/internal/interfaces/console"
)

type ServiceContext struct {
	Ctx    context.Context
	Config *config.Config

	// 基础设施层（第一层初始化）
	apiClients    *factory.APIClients
	wsManager     *websocket.WebSocketManager
	redisClient   *redisclient.Client
	redisRepo     *redisrepo.Repo
	sqliteRepo    *sqliterepo.Repo
	sqliteArbRepo *sqliterepo.ArbitrageRepo

	// 输出端口
	Sink port.Sink

	// 应用业务组件（依赖基础设施）
	priceFeeds            []monitor.PriceFeed
	arbitrageCalculator   *service.ArbitrageCalculator
	arbitrageExecutor     *domainservice.ArbitrageExecutor
	perpetualOrderManager *domainservice.OrderManager

	// 资源管理
	closerChain []func() error
}

// New 创建并初始化 ServiceContext
// 这是应用启动的唯一入口点，所有依赖初始化都在这里完成
func New(ctx context.Context, cfg *config.Config) (*ServiceContext, error) {
	// Initialize exchange clients & registry once (shared across services)
	apiClients, err := factory.NewAPIClients(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("exchange client initialization failed")
		return nil, fmt.Errorf("failed to initialize api clients: %w", err)
	}

	// Initialize WebSocket manager
	wsManager := websocket.NewWebSocketManager()
	if err := wsManager.Initialize(cfg); err != nil {
		log.Fatal().Err(err).Msg("websocket manager initialization failed")
		return nil, fmt.Errorf("failed to initialize websocket manager: %w", err)
	}

	sc := &ServiceContext{
		Ctx:         ctx,
		Config:      cfg,
		apiClients:  apiClients,
		wsManager:   wsManager,
		Sink:        console.NewSink(),
		closerChain: make([]func() error, 0),
	}

	// 初始化所有组件，按依赖顺序
	if err := sc.initializeComponents(); err != nil {
		// 清理已初始化的资源
		_ = sc.Close()
		return nil, err
	}
	return sc, nil
}

// initializeComponents 初始化所有应用组件
// 按照依赖关系有序初始化，确保不会有循环依赖
func (sc *ServiceContext) initializeComponents() error {
	// 0. 初始化存储层 (最基础，最后被其他依赖使用)
	if err := sc.initializeStorage(); err != nil {
		return fmt.Errorf("storage initialization failed: %w", err)
	}
	sc.arbitrageCalculator = service.NewArbitrageCalculator(0.0002) // 默认手续费 0.02%
	sc.arbitrageExecutor = domainservice.NewArbitrageExecutor()

	// 从 WebSocket 管理器中提取 PriceFeed 列表（保持兼容性）
	feeds := extractPriceFeedsFromWSManager(sc.Config.GetEnabledExchanges(), sc.wsManager)
	if len(feeds) == 0 {
		return ErrNoFeedsEnabled
	}
	sc.priceFeeds = feeds
	log.Info().
		Int("feeds", len(feeds)).
		Msg("✓ All components initialized")

	return nil
}

// initializeStorage 初始化存储层 (Redis 和 SQLite)
func (sc *ServiceContext) initializeStorage() error {
	// Redis 初始化
	if sc.Config.Redis.Enabled {
		if err := sc.initRedis(); err != nil {
			return fmt.Errorf("redis initialization failed: %w", err)
		}
	}

	// SQLite 初始化
	if sc.Config.SQLite.Enabled {
		if err := sc.initSQLite(); err != nil {
			return fmt.Errorf("sqlite initialization failed: %w", err)
		}
	}

	// Postgres 初始化 (预留)
	// if sc.Config.Postgres.Enabled {
	// 	if err := sc.initPostgres(); err != nil {
	// 		return fmt.Errorf("postgres initialization failed: %w", err)
	// 	}
	// }

	return nil
}

// initRedis 初始化 Redis 连接
func (sc *ServiceContext) initRedis() error {
	rdb := redisclient.NewClient(&redisclient.Options{
		Addr:     sc.Config.Redis.Addr,
		Password: sc.Config.Redis.Password,
		DB:       sc.Config.Redis.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(sc.Ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	sc.redisClient = rdb
	ttl := time.Duration(sc.Config.Redis.TTLSeconds) * time.Second

	sc.redisRepo = redisrepo.New(
		rdb,
		sc.Config.Redis.Prefix,
		ttl,
		sc.Config.Redis.SignalStream,
		sc.Config.Redis.SignalChannel,
	)

	// 注册关闭回调
	sc.closerChain = append(sc.closerChain, func() error {
		log.Info().Msg("closing redis connection")
		return rdb.Close()
	})

	log.Info().
		Str("addr", sc.Config.Redis.Addr).
		Int("db", sc.Config.Redis.DB).
		Msg("✓ Redis initialized")

	return nil
}

// initSQLite 初始化 SQLite 数据库
func (sc *ServiceContext) initSQLite() error {
	repo, err := sqliterepo.New(sc.Config.SQLite.Path)
	if err != nil {
		return fmt.Errorf("sqlite repo creation failed: %w", err)
	}

	sc.sqliteRepo = repo
	sc.sqliteArbRepo = sqliterepo.NewArbitrageRepo(repo.GetDB())

	// 注册关闭回调
	sc.closerChain = append(sc.closerChain, func() error {
		log.Info().Msg("closing sqlite connection")
		return repo.Close()
	})

	log.Info().
		Str("path", sc.Config.SQLite.Path).
		Msg("✓ SQLite initialized")

	return nil
}

// GetRedisRepo 获取 Redis 仓储
func (sc *ServiceContext) GetRedisRepo() *redisrepo.Repo {
	return sc.redisRepo
}

// GetSQLiteRepo 获取 SQLite 仓储
func (sc *ServiceContext) GetSQLiteRepo() *sqliterepo.Repo {
	return sc.sqliteRepo
}

// BuildMonitorServiceDeps 构建 Monitor Service 所需的所有依赖
// 这个方法由 Application 层 UseCase 调用
// 返回一个完整的、经过验证的依赖集合
func (sc *ServiceContext) BuildMonitorServiceDeps() monitor.ServiceDeps {
	// 获取所有enabled的交易所列表（用于两两比较套利机会）
	enabledExchanges := sc.Config.GetEnabledExchanges()

	return monitor.ServiceDeps{
		Exchanges:      enabledExchanges, // 使用从config中获取的enabled交易所列表
		Feeds:          sc.priceFeeds,
		Symbols:        sc.Config.Symbols.List,
		PrintEveryMin:  sc.Config.App.PrintEveryMin,
		DeltaThreshold: sc.Config.Arbitrage.DeltaThreshold,
		Sink:           sc.Sink,
		Repo:           sc.sqliteRepo,
		ArbitrageRepo:  sc.sqliteArbRepo,
		ArbitrageCalc:  sc.arbitrageCalculator,
		OrderManager:   sc.perpetualOrderManager,
		Executor:       sc.arbitrageExecutor,
	}
}

// GetPriceFeeds 获取已初始化的价格源
func (sc *ServiceContext) GetPriceFeeds() []monitor.PriceFeed {
	return sc.priceFeeds
}

// GetWebSocketManager 获取 WebSocket 管理器
func (sc *ServiceContext) GetWebSocketManager() *websocket.WebSocketManager {
	return sc.wsManager
}

// GetArbitrageCalculator 获取套利计算器
func (sc *ServiceContext) GetArbitrageCalculator() *service.ArbitrageCalculator {
	return sc.arbitrageCalculator
}

// Close 关闭 ServiceContext 中的所有资源
// 包括存储连接、网络连接等
// 应该在应用退出时调用
func (sc *ServiceContext) Close() error {
	// 关闭所有价格源连接
	if sc.priceFeeds != nil {
		for _, feed := range sc.priceFeeds {
			if closeable, ok := feed.(interface{ Close() error }); ok {
				if err := closeable.Close(); err != nil {
					log.Error().Err(err).Msg("error closing price feed")
				}
			}
		}
	}

	// 按照相反的顺序关闭所有资源
	for i := len(sc.closerChain) - 1; i >= 0; i-- {
		if err := sc.closerChain[i](); err != nil {
			log.Error().Err(err).Msg("error closing resource")
		}
	}

	return nil
}

// extractPriceFeedsFromWSManager 从 WebSocket 管理器中提取所有已初始化的 PriceFeed
// 动态遍历所有交易所的现货和合约连接，自动适配新增的交易所
func extractPriceFeedsFromWSManager(enabledExchanges []string, wsm *websocket.WebSocketManager) []monitor.PriceFeed {
	var feeds []monitor.PriceFeed
	for _, exchange := range enabledExchanges {
		// 检查现货 WebSocket 连接
		// if clients := wsm.GetSpotClient(exchange); clients != nil && clients.PriceFeed != nil {
		// 	feeds = append(feeds, clients.PriceFeed)
		// }
		// 检查合约 WebSocket 连接
		if clients := wsm.GetPerpetualClient(exchange); clients != nil && clients.PriceFeed != nil {
			feeds = append(feeds, clients.PriceFeed)
		}
	}

	return feeds
}

// ============================================
// Helper Functions: 构建业务组件
// ============================================

// buildFuturesOrderManager 从 Registry 构建期货订单管理器
// 支持动态初始化所有 enabled 的交易所，用于两两套利对比
// func buildFuturesOrderManager(apiClients *factory.APIClients, cfg *config.Config) (*domainservice.OrderManager, error) {
// 	clients := make(map[string]domainservice.OrderClient)

// 	// Iterate through all exchange configs and initialize perpetual clients for enabled exchanges
// 	for exName, exCfg := range cfg.Exchanges {
// 		if !exCfg.Enabled {
// 			continue // 跳过未启用的交易所
// 		}

// 		exName = strings.ToUpper(exName)

// 		// Check if exchange has perpetual client in Registry
// 		var client domainservice.OrderClient

// 		switch exName {
// 		case "BINANCE":
// 			binanceClients := apiClients.ExchangeRegistry.BinancePerpetual()
// 			if binanceClients != nil && binanceClients.Order != nil {
// 				client = factory.NewBinanceOrderAdapter(binanceClients.Order)
// 				clients["BINANCE"] = client
// 				log.Info().Str("exchange", "BINANCE").Msg("✓ Perpetual order client initialized")
// 			} else {
// 				log.Warn().Str("exchange", "BINANCE").Msg("perpetual order client unavailable")
// 			}

// 		case "BYBIT":
// 			bybitClients := apiClients.ExchangeRegistry.BybitPerpetual()
// 			if bybitClients != nil && bybitClients.Order != nil {
// 				client = factory.NewBybitOrderAdapter(bybitClients.Order)
// 				clients["BYBIT"] = client
// 				log.Info().Str("exchange", "BYBIT").Msg("✓ Perpetual order client initialized")
// 			} else {
// 				log.Warn().Str("exchange", "BYBIT").Msg("perpetual order client unavailable")
// 			}

// 		case "OKX":
// 			// OKX perpetual order adapter not yet implemented
// 			log.Warn().Str("exchange", "OKX").Msg("perpetual order client not yet implemented")

// 		case "BITGET":
// 			// TODO: Implement Bitget perpetual order adapter
// 			log.Warn().Str("exchange", "BITGET").Msg("perpetual order client not yet implemented")

// 		default:
// 			log.Warn().Str("exchange", exName).Msg("unknown exchange, skipping")
// 		}
// 	}

// 	if len(clients) == 0 {
// 		return nil, fmt.Errorf("no perpetual order clients available (no enabled exchanges with order support)")
// 	}

// 	log.Info().
// 		Int("exchanges", len(clients)).
// 		Msg("✓ All enabled perpetual order clients initialized")

// 	// 如果只有两个交易所（向后兼容），使用标准 NewOrderManager
// 	if len(clients) == 2 {
// 		var binanceClient, bybitClient domainservice.OrderClient
// 		if bc, ok := clients["BINANCE"]; ok {
// 			binanceClient = bc
// 		}
// 		if bc, ok := clients["BYBIT"]; ok {
// 			bybitClient = bc
// 		}
// 		return domainservice.NewOrderManager(binanceClient, bybitClient), nil
// 	}

// 	// 如果有多于两个交易所，使用新的构造函数支持所有交易所
// 	return domainservice.NewOrderManagerWithClients(clients), nil
// }
