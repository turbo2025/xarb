package svc

import (
	"context"

	"xarb/internal/application/port"
	"xarb/internal/application/service"
	"xarb/internal/application/usecase/monitor"
	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/container"
	"xarb/internal/infrastructure/factory"
	"xarb/internal/interfaces/console"

	"github.com/rs/zerolog/log"
)

// ServiceContext 参考 go-zero 框架的 ServiceContext 设计模式
// 将应用的所有依赖集中管理在这个结构体中
// 支持完整的组件生命周期管理
type ServiceContext struct {
	Ctx              context.Context
	Config           *config.Config
	Sink             port.Sink
	StorageContainer *container.Container

	// 缓存初始化后的组件，避免重复创建
	priceFeeds            []monitor.PriceFeed
	arbitrageCalculator   *service.ArbitrageCalculator
	symbolMapper          *domainservice.SymbolMapper
	tradeTypeManager      *domainservice.TradeTypeManager
	arbitrageExecutor     *domainservice.ArbitrageExecutor
	futuresOrderManager   *domainservice.OrderManager
	futuresAccountManager *domainservice.AccountManager
}

// New 创建并初始化 ServiceContext
// 这是应用启动的唯一入口点，所有依赖初始化都在这里完成
func New(ctx context.Context, cfg *config.Config) (*ServiceContext, error) {
	storageContainer, err := container.New(cfg)
	if err != nil {
		return nil, err
	}

	sc := &ServiceContext{
		Ctx:              ctx,
		Config:           cfg,
		Sink:             console.NewSink(),
		StorageContainer: storageContainer,
	}

	// 初始化所有组件，按依赖顺序
	if err := sc.initializeComponents(); err != nil {
		return nil, err
	}

	return sc, nil
}

// initializeComponents 初始化所有应用组件
// 按照依赖关系有序初始化，确保不会有循环依赖
func (sc *ServiceContext) initializeComponents() error {
	// 1. 初始化基础组件（无外部依赖）
	sc.arbitrageCalculator = service.NewArbitrageCalculator(0.0002) // 默认手续费 0.02%

	// 2. 初始化 Domain 组件
	sc.symbolMapper = domainservice.NewSymbolMapper()
	if err := sc.symbolMapper.LoadDefaultConfig(); err != nil {
		log.Warn().Err(err).Msg("failed to load default symbol mapping")
	}

	// 3. 初始化 Infrastructure 组件（交易所 API）
	clients := factory.NewAPIClients(sc.Config)
	sc.tradeTypeManager = clients.TradeTypeManager
	sc.arbitrageExecutor = clients.ArbitrageExecutor

	// 4. 初始化订单和账户管理器
	var err1, err2 error
	sc.futuresOrderManager, err1 = sc.tradeTypeManager.GetOrderManager("futures")
	sc.futuresAccountManager, err2 = sc.tradeTypeManager.GetAccountManager("futures")
	if err1 != nil || err2 != nil {
		log.Warn().Msg("failed to get futures clients, continuing with spot only")
	}

	// 5. 初始化价格源（网络连接，需要最后初始化）
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
		ArbitrageRepo:    sc.StorageContainer.SQLiteArbitrageRepo(),
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
	if sc.priceFeeds != nil {
		// 关闭所有价格源连接
		for _, feed := range sc.priceFeeds {
			if closeable, ok := feed.(interface{ Close() error }); ok {
				if err := closeable.Close(); err != nil {
					log.Error().Err(err).Msg("error closing price feed")
				}
			}
		}
	}

	if sc.StorageContainer != nil {
		sc.StorageContainer.Close()
	}

	return nil
}
