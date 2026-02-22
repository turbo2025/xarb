package websocket

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"xarb/internal/application"
	"xarb/internal/application/port"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bitget"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/exchange/okx"
	"xarb/internal/infrastructure/pricefeed"
)

// RetryConfig WebSocket 连接重试配置
type RetryConfig struct {
	MaxRetries int           // 最大重试次数
	InitialDel time.Duration // 初始延迟
	MaxDelay   time.Duration // 最大延迟
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries: 3,
	InitialDel: 1 * time.Second,
	MaxDelay:   10 * time.Second,
}

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
	retryConfig      RetryConfig
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		spotClients:      make(map[string]*WebSocketClients),
		perpetualClients: make(map[string]*WebSocketClients),
		retryConfig:      DefaultRetryConfig,
	}
}

// SetRetryConfig 设置重试配置
func (m *WebSocketManager) SetRetryConfig(cfg RetryConfig) {
	m.retryConfig = cfg
}

// Initialize 初始化已启用交易所的 WebSocket 连接
// 根据配置的 enabled 交易所列表动态初始化价格源和其他流式数据
// 单个交易所失败时继续初始化其他交易所（非关键性失败）
func (m *WebSocketManager) Initialize(cfg *config.Config) error {
	enabledExchanges := cfg.GetEnabledExchanges()
	var failedExchanges []string

	// 获取配置的quote（计价货币）
	quote := strings.TrimSpace(cfg.Symbols.Quote)

	// 为各交易所初始化符号转换器（一次性初始化，避免重复）
	initializeExchangeConverters(quote, enabledExchanges)

	// 定义要初始化的交易类型配置
	type tradeTypeConfig struct {
		name    string
		wsURL   string
		clients map[string]*WebSocketClients
	}

	for _, exchangeName := range enabledExchanges {
		exchCfg := cfg.Exchanges[exchangeName]

		// 遍历所有交易类型（Spot 和 Perpetual），避免重复代码
		configs := []tradeTypeConfig{
			{name: application.TradeTypeSpot, wsURL: exchCfg.SpotWsURL, clients: m.spotClients},
			{name: application.TradeTypePerpetual, wsURL: exchCfg.PerpetualWsURL, clients: m.perpetualClients},
		}

		for _, tc := range configs {
			if tc.wsURL == "" {
				continue
			}
			if err := m.registerWebSocketWithRetry(exchangeName, tc.wsURL, tc.name, tc.clients, quote); err != nil {
				log.Error().Err(err).
					Str("exchange", exchangeName).
					Str("type", tc.name).
					Msg("failed to initialize websocket after retries")
				failedExchanges = append(failedExchanges, exchangeName+":"+tc.name)
				// 继续初始化其他交易所，而不是立即返回错误
			}
		}
	}

	// 如果所有交易所都失败，返回错误
	if len(failedExchanges) == len(enabledExchanges) {
		return fmt.Errorf("failed to initialize websocket for all exchanges: %v", failedExchanges)
	}

	// 如果有部分失败，记录警告但不返回错误
	if len(failedExchanges) > 0 {
		log.Warn().
			Strs("failed_exchanges", failedExchanges).
			Msg("some exchanges failed to initialize websocket, but others succeeded")
	}

	return nil
}

// registerWebSocketWithRetry 带重试的 WebSocket 连接注册
func (m *WebSocketManager) registerWebSocketWithRetry(exchangeName, wsURL, tradeType string, clients map[string]*WebSocketClients, quote string) error {
	var lastErr error
	delay := m.retryConfig.InitialDel

	for attempt := 0; attempt <= m.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Info().
				Str("exchange", exchangeName).
				Str("type", tradeType).
				Int("attempt", attempt).
				Int64("delay_ms", delay.Milliseconds()).
				Msg("retrying websocket connection")
			time.Sleep(delay)
			// 指数退避：每次重试延迟翻倍，但不超过最大延迟
			delay = delay * 2
			if delay > m.retryConfig.MaxDelay {
				delay = m.retryConfig.MaxDelay
			}
		}

		if err := m.registerWebSocket(exchangeName, wsURL, tradeType, clients, quote); err != nil {
			lastErr = err
			continue
		}
		// 成功，直接返回
		return nil
	}

	return fmt.Errorf("failed to register %s %s websocket after %d retries: %w",
		exchangeName, tradeType, m.retryConfig.MaxRetries, lastErr)
}

// registerWebSocket 注册 WebSocket 连接的通用方法
func (m *WebSocketManager) registerWebSocket(exchangeName string, wsURL string, tradeType string, clients map[string]*WebSocketClients, quote string) error {
	factory, ok := pricefeed.Get(exchangeName)
	if !ok {
		return fmt.Errorf("price feed factory not registered for exchange: %s", exchangeName)
	}

	priceFeed := factory(wsURL)
	clients[exchangeName] = &WebSocketClients{PriceFeed: priceFeed}
	log.Info().Str("exchange", exchangeName).Msg("✓ " + exchangeName + " " + tradeType + " websocket initialized")
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

// initializeExchangeConverters 为各交易所初始化符号转换器
// 这样可以避免在每次创建 PriceFeed 时都重新初始化 converter
func initializeExchangeConverters(quote string, enabledExchanges []string) {
	for _, exchangeName := range enabledExchanges {
		switch exchangeName {
		case application.ExchangeOKX:
			okx.InitializeConverter(quote)
		case application.ExchangeBybit:
			bybit.InitializeConverter(quote)
		case application.ExchangeBinance:
			binance.InitializeConverter(quote)
		case application.ExchangeBitget:
			bitget.InitializeConverter(quote)
		}
	}
}
