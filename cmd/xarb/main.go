package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xarb/internal/application/service"
	"xarb/internal/application/usecase/monitor"
	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/container"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bitget"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/exchange/okx"
	"xarb/internal/infrastructure/logger"
	"xarb/internal/interfaces/console"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Setup()

	// Parse flags
	configPath := flag.String("config", "configs/config.toml", "path to config.toml")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config", *configPath).Msg("load config failed")
	}

	// Initialize container with all dependencies
	cont, err := container.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("container initialization failed")
	}
	defer cont.Close()

	// Setup context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize components
	sink := console.NewSink()
	feeds := initializeFeeds(cfg)
	if len(feeds) == 0 {
		log.Fatal().Msg("no exchange feeds enabled")
	}

	// 初始化套利计算器
	arbCalc := service.NewArbitrageCalculator(0.0002) // 默认手续费 0.02%

	// 初始化符号映射器（可选）
	symbolMapper := domainservice.NewSymbolMapper()
	// 加载默认的多交易所、多结算货币配置
	if err := symbolMapper.LoadDefaultConfig(); err != nil {
		log.Warn().Err(err).Msg("failed to load default symbol mapping")
	}

	// 初始化 API 客户端和管理器
	clients := initializeAPIClients(cfg)
	tradeTypeManager := clients.TradeTypeManager
	arbExecutor := clients.ArbitrageExecutor

	// 为了向后兼容，提取期货客户端（如果存在）
	futuresOrderManager, _ := tradeTypeManager.GetOrderManager("futures")
	futuresAccountManager, _ := tradeTypeManager.GetAccountManager("futures")

	// Create service with full arbitrage support
	svc := monitor.NewService(monitor.ServiceDeps{
		Feeds:            feeds,
		Symbols:          cfg.Symbols.List,
		PrintEveryMin:    cfg.App.PrintEveryMin,
		DeltaThreshold:   cfg.Arbitrage.DeltaThreshold,
		Sink:             sink,
		ArbitrageRepo:    cont.SQLiteArbitrageRepo(), // 套利仓储
		ArbitrageCalc:    arbCalc,                    // 套利计算器
		SymbolMapper:     symbolMapper,               // 符号映射器
		OrderManager:     futuresOrderManager,        // 期货订单管理器
		Executor:         arbExecutor,                // 套利执行器
		AccountManager:   futuresAccountManager,      // 期货账户管理器
		TradeTypeManager: tradeTypeManager,           // 交易类型管理器
	})

	// Log startup info
	log.Info().
		Str("config", *configPath).
		Int("symbols", len(cfg.Symbols.List)).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("storage_enabled", cfg.Storage.Enabled).
		Msg("xarb started")

	// Run service
	if err := svc.Run(ctx); err != nil {
		log.Error().Err(err).Msg("monitor service exited")
	}
}

// initializeFeeds 初始化交易所数据源
func initializeFeeds(cfg *config.Config) []monitor.PriceFeed {
	var feeds []monitor.PriceFeed

	if cfg.Exchange.Binance.Enabled {
		feeds = append(feeds, binance.NewFuturesMiniTickerFeed(cfg.Exchange.Binance.WsURL))
		log.Info().Msg("✓ Binance feed initialized")
	} else {
		log.Warn().Msg("⚠️ Binance disabled by config")
	}

	if cfg.Exchange.Bybit.Enabled {
		feeds = append(feeds, bybit.NewLinearTickerFeed(cfg.Exchange.Bybit.WsURL))
		log.Info().Msg("✓ Bybit feed initialized")
	} else {
		log.Warn().Msg("⚠️ Bybit disabled by config")
	}

	if cfg.Exchange.OKX.Enabled {
		feeds = append(feeds, okx.NewPublicLinearTickerFeed(cfg.Exchange.OKX.WsURL))
		log.Info().Msg("✓ OKX feed initialized")
	} else {
		log.Warn().Msg("⚠️ OKX disabled by config")
	}

	if cfg.Exchange.Bitget.Enabled {
		feeds = append(feeds, bitget.NewPublicMarketTickerFeed(cfg.Exchange.Bitget.WsURL))
		log.Info().Msg("✓ Bitget feed initialized")
	} else {
		log.Warn().Msg("⚠️ Bitget disabled by config")
	}

	return feeds
}

// APIClients API 客户端容器
type APIClients struct {
	TradeTypeManager  *domainservice.TradeTypeManager
	ArbitrageExecutor *domainservice.ArbitrageExecutor
}

// initializeAPIClients 初始化所有 API 客户端
func initializeAPIClients(cfg *config.Config) *APIClients {
	tradeTypeManager := domainservice.NewTradeTypeManager()

	// 初始化期货客户端
	futuresOrderMgr, futuresAccountMgr := initializeFuturesClients(cfg)
	tradeTypeManager.SetFuturesClients(futuresOrderMgr, futuresAccountMgr)
	log.Info().Msg("✓ Futures REST API clients initialized")

	// 初始化现货客户端
	spotOrderMgr, spotAccountMgr := initializeSpotClients(cfg)
	tradeTypeManager.SetSpotClients(spotOrderMgr, spotAccountMgr)
	log.Info().Msg("✓ Spot REST API clients initialized")

	return &APIClients{
		TradeTypeManager:  tradeTypeManager,
		ArbitrageExecutor: domainservice.NewArbitrageExecutor(),
	}
}

// initializeFuturesClients 初始化期货客户端
func initializeFuturesClients(cfg *config.Config) (*domainservice.OrderManager, *domainservice.AccountManager) {
	accountManager := domainservice.NewAccountManager(5 * time.Second)

	// 创建 Binance 期货管理器
	binanceFuturesMgr := binance.NewFuturesManager(
		cfg.Exchange.Binance.APIKey,
		cfg.Exchange.Binance.SecretKey,
		cfg.Exchange.Binance.FuturesURL,
	)

	// 创建 Bybit 线性期货管理器
	bybitLinearMgr := bybit.NewLinearManager(
		cfg.Exchange.Bybit.APIKey,
		cfg.Exchange.Bybit.SecretKey,
		cfg.Exchange.Bybit.LinearURL,
	)

	// 注册账户客户端
	accountManager.RegisterClient("binance", binanceFuturesMgr.Account)
	accountManager.RegisterClient("bybit", bybitLinearMgr.Account)

	// 创建订单管理器
	orderManager := domainservice.NewOrderManager(
		newBinanceOrderAdapter(binanceFuturesMgr.Order),
		newBybitOrderAdapter(bybitLinearMgr.Order),
	)

	return orderManager, accountManager
}

// initializeSpotClients 初始化现货客户端
func initializeSpotClients(cfg *config.Config) (*domainservice.OrderManager, *domainservice.AccountManager) {
	accountManager := domainservice.NewAccountManager(5 * time.Second)

	// 创建 Binance 现货管理器
	binanceSpotMgr := binance.NewSpotManager(
		cfg.Exchange.Binance.APIKey,
		cfg.Exchange.Binance.SecretKey,
		cfg.Exchange.Binance.SpotURL,
	)

	// 创建 Bybit 现货管理器
	bybitSpotMgr := bybit.NewSpotManager(
		cfg.Exchange.Bybit.APIKey,
		cfg.Exchange.Bybit.SecretKey,
		cfg.Exchange.Bybit.SpotURL,
	)

	// 注册账户客户端
	accountManager.RegisterClient("binance", binanceSpotMgr.Account)
	accountManager.RegisterClient("bybit", bybitSpotMgr.Account)

	// 创建订单管理器
	orderManager := domainservice.NewOrderManager(
		newBinanceSpotOrderAdapter(binanceSpotMgr.Order),
		newBybitSpotOrderAdapter(bybitSpotMgr.Order),
	)

	return orderManager, accountManager
}

// 订单适配器
// newBinanceOrderAdapter 创建 Binance 订单客户端适配器
func newBinanceOrderAdapter(client *binance.FuturesOrderClient) domainservice.OrderClient {
	return &binanceOrderClientAdapter{client: client}
}

// binanceOrderClientAdapter 适配 Binance 客户端为 OrderClient 接口
type binanceOrderClientAdapter struct {
	client *binance.FuturesOrderClient
}

func (a *binanceOrderClientAdapter) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error) {
	return a.client.PlaceOrder(ctx, symbol, side, quantity, price, isMarket)
}

func (a *binanceOrderClientAdapter) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	return a.client.CancelOrder(ctx, symbol, orderId)
}

func (a *binanceOrderClientAdapter) GetOrderStatus(ctx context.Context, symbol string, orderId string) (*domainservice.OrderStatus, error) {
	status, err := a.client.GetOrderStatus(ctx, symbol, orderId)
	if err != nil {
		return nil, err
	}
	return &domainservice.OrderStatus{
		OrderID:          status.OrderID,
		Symbol:           status.Symbol,
		Side:             status.Side,
		Quantity:         status.Quantity,
		ExecutedQuantity: status.ExecutedQuantity,
		Price:            status.Price,
		AvgExecutedPrice: status.AvgExecutedPrice,
		Status:           status.Status,
		CreatedAt:        status.CreatedAt,
		UpdatedAt:        status.UpdatedAt,
	}, nil
}

func (a *binanceOrderClientAdapter) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	return a.client.GetFundingRate(ctx, symbol)
}

// newBybitOrderAdapter 创建 Bybit 订单客户端适配器
func newBybitOrderAdapter(client *bybit.LinearOrderClient) domainservice.OrderClient {
	return &bybitOrderClientAdapter{client: client}
}

// 现货订单适配器
// newBinanceSpotOrderAdapter 创建 Binance 现货订单客户端适配器
func newBinanceSpotOrderAdapter(client *binance.SpotOrderClient) domainservice.OrderClient {
	return &binanceSpotOrderClientAdapter{client: client}
}

// binanceSpotOrderClientAdapter 适配 Binance 现货客户端为 OrderClient 接口
type binanceSpotOrderClientAdapter struct {
	client *binance.SpotOrderClient
}

func (a *binanceSpotOrderClientAdapter) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error) {
	return a.client.PlaceOrder(ctx, symbol, side, quantity, price, isMarket)
}

func (a *binanceSpotOrderClientAdapter) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	return a.client.CancelOrder(ctx, symbol, orderId)
}

func (a *binanceSpotOrderClientAdapter) GetOrderStatus(ctx context.Context, symbol string, orderId string) (*domainservice.OrderStatus, error) {
	status, err := a.client.GetOrderStatus(ctx, symbol, orderId)
	if err != nil {
		return nil, err
	}
	return &domainservice.OrderStatus{
		OrderID:          status.OrderID,
		Symbol:           status.Symbol,
		Side:             status.Side,
		Quantity:         status.Quantity,
		ExecutedQuantity: status.ExecutedQuantity,
		Price:            status.Price,
		AvgExecutedPrice: status.AvgExecutedPrice,
		Status:           status.Status,
		CreatedAt:        status.CreatedAt,
		UpdatedAt:        status.UpdatedAt,
	}, nil
}

func (a *binanceSpotOrderClientAdapter) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	return a.client.GetFundingRate(ctx, symbol)
}

// newBybitSpotOrderAdapter 创建 Bybit 现货订单客户端适配器
func newBybitSpotOrderAdapter(client *bybit.SpotOrderClient) domainservice.OrderClient {
	return &bybitSpotOrderClientAdapter{client: client}
}

// bybitSpotOrderClientAdapter 适配 Bybit 现货客户端为 OrderClient 接口
type bybitSpotOrderClientAdapter struct {
	client *bybit.SpotOrderClient
}

func (a *bybitSpotOrderClientAdapter) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error) {
	return a.client.PlaceOrder(ctx, symbol, side, quantity, price, isMarket)
}

func (a *bybitSpotOrderClientAdapter) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	return a.client.CancelOrder(ctx, symbol, orderId)
}

func (a *bybitSpotOrderClientAdapter) GetOrderStatus(ctx context.Context, symbol string, orderId string) (*domainservice.OrderStatus, error) {
	status, err := a.client.GetOrderStatus(ctx, symbol, orderId)
	if err != nil {
		return nil, err
	}
	return &domainservice.OrderStatus{
		OrderID:          status.OrderID,
		Symbol:           status.Symbol,
		Side:             status.Side,
		Quantity:         status.Quantity,
		ExecutedQuantity: status.ExecutedQuantity,
		Price:            status.Price,
		AvgExecutedPrice: status.AvgExecutedPrice,
		Status:           status.Status,
		CreatedAt:        status.CreatedAt,
		UpdatedAt:        status.UpdatedAt,
	}, nil
}

func (a *bybitSpotOrderClientAdapter) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	return a.client.GetFundingRate(ctx, symbol)
}

// bybitOrderClientAdapter 适配 Bybit 客户端为 OrderClient 接口
type bybitOrderClientAdapter struct {
	client *bybit.LinearOrderClient
}

func (a *bybitOrderClientAdapter) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64, isMarket bool) (string, error) {
	return a.client.PlaceOrder(ctx, symbol, side, quantity, price, isMarket)
}

func (a *bybitOrderClientAdapter) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	return a.client.CancelOrder(ctx, symbol, orderId)
}

func (a *bybitOrderClientAdapter) GetOrderStatus(ctx context.Context, symbol string, orderId string) (*domainservice.OrderStatus, error) {
	status, err := a.client.GetOrderStatus(ctx, symbol, orderId)
	if err != nil {
		return nil, err
	}
	return &domainservice.OrderStatus{
		OrderID:          status.OrderID,
		Symbol:           status.Symbol,
		Side:             status.Side,
		Quantity:         status.Quantity,
		ExecutedQuantity: status.ExecutedQuantity,
		Price:            status.Price,
		AvgExecutedPrice: status.AvgExecutedPrice,
		Status:           status.Status,
		CreatedAt:        status.CreatedAt,
		UpdatedAt:        status.UpdatedAt,
	}, nil
}

func (a *bybitOrderClientAdapter) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	return a.client.GetFundingRate(ctx, symbol)
}
