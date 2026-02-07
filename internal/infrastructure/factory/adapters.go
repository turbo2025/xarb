package factory

import (
	"context"

	domainservice "xarb/internal/domain/service"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
)

// ============================================
// Binance perpetual order adapter
// ============================================

type binanceOrderClientAdapter struct {
	client *binance.PerpetualOrderClient
}

func NewBinanceOrderAdapter(client *binance.PerpetualOrderClient) domainservice.OrderClient {
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

// ============================================
// Bybit perpetual order adapter
// ============================================

type bybitOrderClientAdapter struct {
	client *bybit.PerpetualOrderClient
}

func NewBybitOrderAdapter(client *bybit.PerpetualOrderClient) domainservice.OrderClient {
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

// ============================================
// Binance 现货订单适配器
// ============================================

type binanceSpotOrderClientAdapter struct {
	client *binance.SpotOrderClient
}

func NewBinanceSpotOrderAdapter(client *binance.SpotOrderClient) domainservice.OrderClient {
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

// ============================================
// Bybit 现货订单适配器
// ============================================

type bybitSpotOrderClientAdapter struct {
	client *bybit.SpotOrderClient
}

func NewBybitSpotOrderAdapter(client *bybit.SpotOrderClient) domainservice.OrderClient {
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
