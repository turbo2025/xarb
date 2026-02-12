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
	"xarb/internal/interfaces/console"
)

// ServiceContext 参考 go-zero 框架的 ServiceContext 设计模式
// 将应用的所有依赖集中管理在这个结构体中
// 支持完整的组件生命周期管理
type ServiceContext struct {
	Ctx context.Context

	// 配置
	Config *config.Config
	// 交易所客户端
	apiClients *factory.APIClients

	// 输出端口
	Sink port.Sink

	// 存储层组件
	redisClient   *redisclient.Client
	redisRepo     *redisrepo.Repo
	sqliteRepo    *sqliterepo.Repo
	sqliteArbRepo *sqliterepo.ArbitrageRepo

	// 应用业务组件
	priceFeeds            []monitor.PriceFeed
	arbitrageCalculator   *service.ArbitrageCalculator
	symbolMapper          *domainservice.SymbolMapper
	tradeTypeManager      *domainservice.TradeTypeManager
	arbitrageExecutor     *domainservice.ArbitrageExecutor
	futuresOrderManager   *domainservice.OrderManager
	spotOrderManager      *domainservice.OrderManager
	futuresAccountManager *domainservice.AccountManager

	// 资源清理链
	closerChain []func() error
}

// New 创建并初始化 ServiceContext
// 这是应用启动的唯一入口点，所有依赖初始化都在这里完成
func New(ctx context.Context, cfg *config.Config, apiClients *factory.APIClients) (*ServiceContext, error) {
	sc := &ServiceContext{
		Ctx:         ctx,
		Config:      cfg,
		Sink:        console.NewSink(),
		apiClients:  apiClients,
		closerChain: make([]func() error, 0),
	}

	// 如果未传入 API 客户端，则在这里初始化一次
	if sc.apiClients == nil {
		clients, err := factory.NewAPIClients(sc.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize api clients: %w", err)
		}
		sc.apiClients = clients
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
	if sc.Config.Storage.Enabled {
		if err := sc.initializeStorage(); err != nil {
			return fmt.Errorf("storage initialization failed: %w", err)
		}
	}

	// 1. 初始化基础组件（无外部依赖）
	sc.arbitrageCalculator = service.NewArbitrageCalculator(0.0002) // 默认手续费 0.02%

	// 2. 初始化 Domain 组件
	sc.symbolMapper = domainservice.NewSymbolMapper()
	if err := sc.symbolMapper.LoadDefaultConfig(); err != nil {
		log.Warn().Err(err).Msg("failed to load default symbol mapping")
	}

	if sc.apiClients == nil {
		return fmt.Errorf("api clients not initialized")
	}

	// 4. 初始化业务组件
	sc.arbitrageExecutor = domainservice.NewArbitrageExecutor()

	// 5. 从 ExchangeRegistry 构建所需的 Manager
	futuresOrderMgr, err := buildFuturesOrderManager(sc.apiClients)
	if err != nil {
		log.Warn().Err(err).Msg("failed to build futures order manager")
	} else {
		sc.futuresOrderManager = futuresOrderMgr
	}

	spotOrderMgr, err := buildSpotOrderManager(sc.apiClients)
	if err != nil {
		log.Warn().Err(err).Msg("failed to build spot order manager")
	} else {
		sc.spotOrderManager = spotOrderMgr
	}

	// 6. 初始化价格源（网络连接，需要最后初始化）
	feeds := factory.NewPriceFeeds(sc.Config)
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
	if sc.Config.Storage.Redis.Enabled {
		if err := sc.initRedis(); err != nil {
			return fmt.Errorf("redis initialization failed: %w", err)
		}
	}

	// SQLite 初始化
	if sc.Config.Storage.SQLite.Enabled {
		if err := sc.initSQLite(); err != nil {
			return fmt.Errorf("sqlite initialization failed: %w", err)
		}
	}

	// Postgres 初始化 (预留)
	// if sc.Config.Storage.Postgres.Enabled {
	// 	if err := sc.initPostgres(); err != nil {
	// 		return fmt.Errorf("postgres initialization failed: %w", err)
	// 	}
	// }

	return nil
}

// initRedis 初始化 Redis 连接
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

	// 注册关闭回调
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

// initSQLite 初始化 SQLite 数据库
func (sc *ServiceContext) initSQLite() error {
	repo, err := sqliterepo.New(sc.Config.Storage.SQLite.Path)
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
		Str("path", sc.Config.Storage.SQLite.Path).
		Msg("✓ SQLite initialized")

	return nil
}

// GetSQLiteRepo 获取 SQLite 仓储
func (sc *ServiceContext) GetSQLiteRepo() *sqliterepo.Repo {
	return sc.sqliteRepo
}

// GetRedisRepo 获取 Redis 仓储
func (sc *ServiceContext) GetRedisRepo() *redisrepo.Repo {
	return sc.redisRepo
}

// BuildMonitorServiceDeps 构建 Monitor Service 所需的所有依赖
// 这个方法由 Application 层 UseCase 调用
// 返回一个完整的、经过验证的依赖集合
func (sc *ServiceContext) BuildMonitorServiceDeps() monitor.ServiceDeps {
	return monitor.ServiceDeps{
		Feeds:            sc.priceFeeds,
		Symbols:          sc.Config.Symbols.List,
		PrintEveryMin:    sc.Config.App.PrintEveryMin,
		DeltaThreshold:   sc.Config.Arbitrage.DeltaThreshold,
		Sink:             sc.Sink,
		ArbitrageRepo:    sc.sqliteArbRepo,
		ArbitrageCalc:    sc.arbitrageCalculator,
		SymbolMapper:     sc.symbolMapper,
		OrderManager:     sc.futuresOrderManager,
		Executor:         sc.arbitrageExecutor,
		AccountManager:   sc.futuresAccountManager,
		TradeTypeManager: sc.tradeTypeManager,
	}
}

// GetPriceFeeds 获取已初始化的价格源
func (sc *ServiceContext) GetPriceFeeds() []monitor.PriceFeed {
	return sc.priceFeeds
}

// GetArbitrageCalculator 获取套利计算器
func (sc *ServiceContext) GetArbitrageCalculator() *service.ArbitrageCalculator {
	return sc.arbitrageCalculator
}

// GetSymbolMapper 获取符号映射器
func (sc *ServiceContext) GetSymbolMapper() *domainservice.SymbolMapper {
	return sc.symbolMapper
}

// GetTradeTypeManager 获取交易类型管理器
func (sc *ServiceContext) GetTradeTypeManager() *domainservice.TradeTypeManager {
	return sc.tradeTypeManager
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

// ============================================
// Helper Functions: 构建业务组件
// ============================================

// buildFuturesOrderManager 从 Registry 构建期货订单管理器
func buildFuturesOrderManager(apiClients *factory.APIClients) (*domainservice.OrderManager, error) {
	binanceClients := apiClients.ExchangeRegistry.BinanceFutures()
	bybitClients := apiClients.ExchangeRegistry.BybitFutures()

	var binanceAdapter domainservice.OrderClient
	var bybitAdapter domainservice.OrderClient

	if binanceClients != nil && binanceClients.Order != nil {
		binanceAdapter = factory.NewBinanceOrderAdapter(binanceClients.Order)
	}

	if bybitClients != nil && bybitClients.Order != nil {
		bybitAdapter = factory.NewBybitOrderAdapter(bybitClients.Order)
	}

	if binanceAdapter == nil && bybitAdapter == nil {
		return nil, fmt.Errorf("no futures order clients available")
	}

	return domainservice.NewOrderManager(binanceAdapter, bybitAdapter), nil
}

// buildSpotOrderManager 从 Registry 构建现货订单管理器
func buildSpotOrderManager(apiClients *factory.APIClients) (*domainservice.OrderManager, error) {
	binanceClients := apiClients.ExchangeRegistry.BinanceSpot()
	bybitClients := apiClients.ExchangeRegistry.BybitSpot()

	var binanceAdapter domainservice.OrderClient
	var bybitAdapter domainservice.OrderClient

	if binanceClients != nil && binanceClients.Order != nil {
		binanceAdapter = factory.NewBinanceSpotOrderAdapter(binanceClients.Order)
	}

	if bybitClients != nil && bybitClients.Order != nil {
		bybitAdapter = factory.NewBybitSpotOrderAdapter(bybitClients.Order)
	}

	if binanceAdapter == nil && bybitAdapter == nil {
		return nil, fmt.Errorf("no spot order clients available")
	}

	return domainservice.NewOrderManager(binanceAdapter, bybitAdapter), nil
}
