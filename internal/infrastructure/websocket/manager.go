package websocket

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"xarb/internal/application/port"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/pricefeed"
)

// WebSocketClients 通用的 WebSocket 客户端集合（支持扩展）
type WebSocketClients struct {
	PriceFeed port.PriceFeed
	// OrderBook port.OrderBook // 未来添加
}

// WebSocketManager 统一管理所有交易所的 WebSocket 连接（价格源、订单簿等）
// 使用 map 存储，支持动态扩展而无需修改结构体
type WebSocketManager struct {
	spotClients      map[string]*WebSocketClients
	perpetualClients map[string]*WebSocketClients
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		spotClients:      make(map[string]*WebSocketClients),
		perpetualClients: make(map[string]*WebSocketClients),
	}
}

// Initialize 初始化已启用交易所的 WebSocket 连接
// 根据配置的 enabled 交易所列表动态初始化价格源和其他流式数据
func (m *WebSocketManager) Initialize(cfg *config.Config) error {
	enabledExchanges := cfg.GetEnabledExchanges()

	for _, exchangeName := range enabledExchanges {
		exchCfg := cfg.Exchanges[exchangeName]

		// 初始化 Spot WebSocket 连接（如果配置了 SpotWsURL）
		// if exchCfg.SpotWsURL != "" {
		// 	if err := m.registerSpotWebSocket(exchangeName, &exchCfg); err != nil {
		// 		log.Error().Err(err).Str("exchange", exchangeName).Msg("failed to initialize spot websocket")
		// 		return err
		// 	}
		// }

		// 初始化 Perpetual WebSocket 连接（如果配置了 PerpetualWsURL）
		if exchCfg.PerpetualWsURL != "" {
			if err := m.registerPerpetualWebSocket(exchangeName, &exchCfg); err != nil {
				log.Error().Err(err).Str("exchange", exchangeName).Msg("failed to initialize perpetual websocket")
				return err
			}
		}
	}

	return nil
}

// registerSpotWebSocket 注册 Spot WebSocket 连接
func (m *WebSocketManager) registerSpotWebSocket(exchangeName string, cfg *config.ExchangeConfig) error {
	factory, ok := pricefeed.Get(exchangeName)
	if !ok {
		return fmt.Errorf("price feed factory not registered for exchange: %s", exchangeName)
	}

	priceFeed := factory(cfg.SpotWsURL)
	m.spotClients[exchangeName] = &WebSocketClients{PriceFeed: priceFeed}
	log.Info().Str("exchange", exchangeName).Msg("✓ " + exchangeName + " spot websocket initialized")
	return nil
}

// registerPerpetualWebSocket 注册 Perpetual WebSocket 连接
func (m *WebSocketManager) registerPerpetualWebSocket(exchangeName string, cfg *config.ExchangeConfig) error {
	factory, ok := pricefeed.Get(exchangeName)
	if !ok {
		return fmt.Errorf("price feed factory not registered for exchange: %s", exchangeName)
	}

	priceFeed := factory(cfg.PerpetualWsURL)
	m.perpetualClients[exchangeName] = &WebSocketClients{PriceFeed: priceFeed}
	log.Info().Str("exchange", exchangeName).Msg("✓ " + exchangeName + " perpetual websocket initialized")
	return nil
}

// Getter 方法

// GetSpotClient 获取指定交易所的现货 WebSocket 客户端
func (m *WebSocketManager) GetSpotClient(exchangeName string) *WebSocketClients {
	return m.spotClients[exchangeName]
}

// GetPerpetualClient 获取指定交易所的永续合约 WebSocket 客户端
func (m *WebSocketManager) GetPerpetualClient(exchangeName string) *WebSocketClients {
	return m.perpetualClients[exchangeName]
}
