package factory

import (
	"context"
	"fmt"

	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"

	"github.com/rs/zerolog/log"
)

// APIClients API 客户端容器
// 职责: 只管理交易所客户端的初始化和注册
type APIClients struct {
	ExchangeRegistry *ExchangeClientRegistry
}

// NewAPIClients 初始化所有交易所客户端
// 策略: 动态遍历 cfg.Exchanges 并注册所有启用的交易所
func NewAPIClients(cfg *config.Config) *APIClients {
	registry := NewExchangeClientRegistry()

	// 动态注册已启用的交易所
	registerExchanges(registry, cfg)

	return &APIClients{
		ExchangeRegistry: registry,
	}
}

// registerExchanges 根据配置动态注册所有启用的交易所
func registerExchanges(registry *ExchangeClientRegistry, cfg *config.Config) {
	// Binance
	if cfg.Exchanges.Binance.Enabled {
		registry.RegisterBinance(
			cfg.Exchanges.Binance.APIKey,
			cfg.Exchanges.Binance.SecretKey,
			cfg.Exchanges.Binance.FuturesURL,
			cfg.Exchanges.Binance.SpotURL,
		)
		log.Info().Msg("✓ Binance clients registered (Spot + Futures)")
	}

	// Bybit
	if cfg.Exchanges.Bybit.Enabled {
		registry.RegisterBybit(
			cfg.Exchanges.Bybit.APIKey,
			cfg.Exchanges.Bybit.SecretKey,
			cfg.Exchanges.Bybit.FuturesURL,
			cfg.Exchanges.Bybit.SpotURL,
		)
		log.Info().Msg("✓ Bybit clients registered (Spot + Futures)")
	}

	// OKX (预留)
	// if cfg.Exchanges.OKX.Enabled {
	// 	registry.RegisterOKX(...)
	// 	log.Info().Msg("✓ OKX clients registered (Spot + Futures)")
	// }

	// Bitget (预留)
	// if cfg.Exchanges.Bitget.Enabled {
	// 	registry.RegisterBitget(...)
	// 	log.Info().Msg("✓ Bitget clients registered (Spot + Futures)")
	// }
}

// ============================================
// 辅助函数: 为 ServiceContext 构建所需的 Manager
// ============================================

// BuildFuturesOrderManager 从 Registry 构建期货订单管理器
func (api *APIClients) BuildFuturesOrderManager() (*domainservice.OrderManager, error) {
	binanceClients, binanceErr := api.ExchangeRegistry.GetBizSet(ExchangeBinance, TradeTypeFutures)
	bybitClients, bybitErr := api.ExchangeRegistry.GetBizSet(ExchangeBybit, TradeTypeFutures)

	var binanceAdapter domainservice.OrderClient
	var bybitAdapter domainservice.OrderClient

	if binanceErr == nil && binanceClients != nil {
		if futuresOrder, ok := binanceClients.Order.(*binance.FuturesOrderClient); ok {
			binanceAdapter = newBinanceOrderAdapter(futuresOrder)
		}
	}

	if bybitErr == nil && bybitClients != nil {
		if linearOrder, ok := bybitClients.Order.(*bybit.LinearOrderClient); ok {
			bybitAdapter = newBybitOrderAdapter(linearOrder)
		}
	}

	if binanceAdapter == nil && bybitAdapter == nil {
		return nil, fmt.Errorf("no futures order clients available")
	}

	return domainservice.NewOrderManager(binanceAdapter, bybitAdapter), nil
}

// BuildSpotOrderManager 从 Registry 构建现货订单管理器
func (api *APIClients) BuildSpotOrderManager() (*domainservice.OrderManager, error) {
	binanceClients, binanceErr := api.ExchangeRegistry.GetBizSet(ExchangeBinance, TradeTypeSpot)
	bybitClients, bybitErr := api.ExchangeRegistry.GetBizSet(ExchangeBybit, TradeTypeSpot)

	var binanceAdapter domainservice.OrderClient
	var bybitAdapter domainservice.OrderClient

	if binanceErr == nil && binanceClients != nil {
		if spotOrder, ok := binanceClients.Order.(*binance.SpotOrderClient); ok {
			binanceAdapter = newBinanceSpotOrderAdapter(spotOrder)
		}
	}

	if bybitErr == nil && bybitClients != nil {
		if spotOrder, ok := bybitClients.Order.(*bybit.SpotOrderClient); ok {
			bybitAdapter = newBybitSpotOrderAdapter(spotOrder)
		}
	}

	if binanceAdapter == nil && bybitAdapter == nil {
		return nil, fmt.Errorf("no spot order clients available")
	}

	return domainservice.NewOrderManager(binanceAdapter, bybitAdapter), nil
}

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

// Binance 现货订单适配器（复用期货适配器逻辑）
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
	// Spot trading does not have funding rates
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
