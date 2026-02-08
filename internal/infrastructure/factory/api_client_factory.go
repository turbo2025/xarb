package factory

import (
	"context"
	"time"

	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"

	"github.com/rs/zerolog/log"
)

// APIClients API 客户端容器
type APIClients struct {
	TradeTypeManager  *domainservice.TradeTypeManager
	ArbitrageExecutor *domainservice.ArbitrageExecutor
}

// NewAPIClients 初始化所有 API 客户端
func NewAPIClients(cfg *config.Config) *APIClients {
	tradeTypeManager := domainservice.NewTradeTypeManager()

	// 初始化期货客户端
	futuresOrderMgr, futuresAccountMgr := newFuturesClients(cfg)
	tradeTypeManager.SetFuturesClients(futuresOrderMgr, futuresAccountMgr)
	log.Info().Msg("✓ Futures REST API clients initialized")

	// 初始化现货客户端
	spotOrderMgr, spotAccountMgr := newSpotClients(cfg)
	tradeTypeManager.SetSpotClients(spotOrderMgr, spotAccountMgr)
	log.Info().Msg("✓ Spot REST API clients initialized")

	return &APIClients{
		TradeTypeManager:  tradeTypeManager,
		ArbitrageExecutor: domainservice.NewArbitrageExecutor(),
	}
}

// newFuturesClients 初始化期货客户端
func newFuturesClients(cfg *config.Config) (*domainservice.OrderManager, *domainservice.AccountManager) {
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

// newSpotClients 初始化现货客户端
func newSpotClients(cfg *config.Config) (*domainservice.OrderManager, *domainservice.AccountManager) {
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

// Binance 期货订单适配器
type binanceOrderClientAdapter struct {
	client *binance.FuturesOrderClient
}

func newBinanceOrderAdapter(client *binance.FuturesOrderClient) domainservice.OrderClient {
	return &binanceOrderClientAdapter{client: client}
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

// Bybit 期货订单适配器
type bybitOrderClientAdapter struct {
	client *bybit.LinearOrderClient
}

func newBybitOrderAdapter(client *bybit.LinearOrderClient) domainservice.OrderClient {
	return &bybitOrderClientAdapter{client: client}
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

// Binance 现货订单适配器
type binanceSpotOrderClientAdapter struct {
	client *binance.SpotOrderClient
}

func newBinanceSpotOrderAdapter(client *binance.SpotOrderClient) domainservice.OrderClient {
	return &binanceSpotOrderClientAdapter{client: client}
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

// Bybit 现货订单适配器
type bybitSpotOrderClientAdapter struct {
	client *bybit.SpotOrderClient
}

func newBybitSpotOrderAdapter(client *bybit.SpotOrderClient) domainservice.OrderClient {
	return &bybitSpotOrderClientAdapter{client: client}
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
