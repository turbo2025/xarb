package service

import (
	"fmt"
)

// TradeTypeManager 交易类型管理器，支持期货和现货
type TradeTypeManager struct {
	futures struct {
		orderManager   *OrderManager
		accountManager *AccountManager
	}
	spot struct {
		orderManager   *OrderManager
		accountManager *AccountManager
	}
}

// NewTradeTypeManager 创建交易类型管理器
func NewTradeTypeManager() *TradeTypeManager {
	return &TradeTypeManager{}
}

// SetFuturesClients 设置期货客户端
func (tm *TradeTypeManager) SetFuturesClients(orderManager *OrderManager, accountManager *AccountManager) {
	tm.futures.orderManager = orderManager
	tm.futures.accountManager = accountManager
}

// SetSpotClients 设置现货客户端
func (tm *TradeTypeManager) SetSpotClients(orderManager *OrderManager, accountManager *AccountManager) {
	tm.spot.orderManager = orderManager
	tm.spot.accountManager = accountManager
}

// GetOrderManager 获取订单管理器
func (tm *TradeTypeManager) GetOrderManager(tradeType string) (*OrderManager, error) {
	switch tradeType {
	case "futures":
		if tm.futures.orderManager == nil {
			return nil, fmt.Errorf("futures order manager not initialized")
		}
		return tm.futures.orderManager, nil
	case "spot":
		if tm.spot.orderManager == nil {
			return nil, fmt.Errorf("spot order manager not initialized")
		}
		return tm.spot.orderManager, nil
	default:
		return nil, fmt.Errorf("unknown trade type: %s", tradeType)
	}
}

// GetAccountManager 获取账户管理器
func (tm *TradeTypeManager) GetAccountManager(tradeType string) (*AccountManager, error) {
	switch tradeType {
	case "futures":
		if tm.futures.accountManager == nil {
			return nil, fmt.Errorf("futures account manager not initialized")
		}
		return tm.futures.accountManager, nil
	case "spot":
		if tm.spot.accountManager == nil {
			return nil, fmt.Errorf("spot account manager not initialized")
		}
		return tm.spot.accountManager, nil
	default:
		return nil, fmt.Errorf("unknown trade type: %s", tradeType)
	}
}

// HasFutures 检查是否已初始化期货客户端
func (tm *TradeTypeManager) HasFutures() bool {
	return tm.futures.orderManager != nil && tm.futures.accountManager != nil
}

// HasSpot 检查是否已初始化现货客户端
func (tm *TradeTypeManager) HasSpot() bool {
	return tm.spot.orderManager != nil && tm.spot.accountManager != nil
}

// GetAvailableTradeTypes 获取可用的交易类型
func (tm *TradeTypeManager) GetAvailableTradeTypes() []string {
	var types []string
	if tm.HasFutures() {
		types = append(types, "futures")
	}
	if tm.HasSpot() {
		types = append(types, "spot")
	}
	return types
}
