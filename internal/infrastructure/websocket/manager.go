package websocket

import (
	"fmt"
	"reflect"
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
// 使用嵌套 map 存储，支持每个交易所同时有多种交易类型（Spot/Perpetual）
type WebSocketManager struct {
	// clients[exchangeName][tradeType] = *WebSocketClients
	clients     map[string]map[string]*WebSocketClients
	retryConfig RetryConfig
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[string]map[string]*WebSocketClients),
		retryConfig: DefaultRetryConfig,
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

	// 为各交易所初始化符号转换器（一次性初始化，避免重复）
	initializeExchangeConverters(strings.TrimSpace(cfg.Symbols.Quote), enabledExchanges)

	for _, exchangeName := range enabledExchanges {
		// 遍历不同交易类型（Spot 和 Perpetual）
		// 使用反射动态获取ExchangeConfig中的TradeConfig字段
		tradeConfigs := []struct {
			name   string
			config config.TradeConfig
		}{}

		exchCfg := cfg.Exchanges[exchangeName]

		// 使用反射遍历ExchangeConfig的所有字段
		exchType := reflect.TypeOf(exchCfg)
		exchValue := reflect.ValueOf(exchCfg)

		for i := 0; i < exchType.NumField(); i++ {
			field := exchType.Field(i)
			// 检查字段是否是TradeConfig类型
			if field.Type == reflect.TypeOf(config.TradeConfig{}) {
				fieldName := strings.ToLower(field.Name)
				fieldValue := exchValue.Field(i).Interface().(config.TradeConfig)

				// 只添加配置中启用且有效的交易类型
				if fieldValue.Enabled && strings.TrimSpace(fieldValue.WS) != "" {
					tradeConfigs = append(tradeConfigs, struct {
						name   string
						config config.TradeConfig
					}{
						name:   fieldName,
						config: fieldValue,
					})
				}
			}
		}

		for _, tc := range tradeConfigs {
			// 检查enabled标志和WebSocket URL
			if !tc.config.Enabled {
				log.Debug().Str("exchange", exchangeName).Str("type", tc.name).Msg("websocket disabled in config")
				continue
			}
			if strings.TrimSpace(tc.config.WS) == "" {
				continue
			}
			if err := m.registerWebSocketWithRetry(exchangeName, tc.config.WS, tc.name); err != nil {
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
func (m *WebSocketManager) registerWebSocketWithRetry(exchangeName, wsURL, tradeType string) error {
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

		if err := m.registerWebSocket(exchangeName, wsURL, tradeType); err != nil {
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
// tradeType: "spot" 或 "perpetual"
func (m *WebSocketManager) registerWebSocket(exchangeName string, wsURL string, tradeType string) error {
	factory, ok := pricefeed.Get(exchangeName)
	if !ok {
		return fmt.Errorf("price feed factory not registered for exchange: %s", exchangeName)
	}

	// 创建 PriceFeed 时传递 tradeType，让工厂正确处理 spot 还是 perpetual
	priceFeed := factory(wsURL, tradeType)

	// 初始化该交易所的map（如果还未初始化）
	if m.clients[exchangeName] == nil {
		m.clients[exchangeName] = make(map[string]*WebSocketClients)
	}

	// 在该交易所的map中存储该交易类型的客户端
	m.clients[exchangeName][tradeType] = &WebSocketClients{PriceFeed: priceFeed}
	log.Info().
		Str("exchange", exchangeName).
		Str("type", tradeType).
		Msg("✓ websocket initialized")
	return nil
}

// Getter 方法

// GetClient 获取指定交易所和交易类型的 WebSocket 客户端
// tradeType: "spot" 或 "perpetual"
func (m *WebSocketManager) GetClient(exchangeName, tradeType string) *WebSocketClients {
	if exchange, ok := m.clients[exchangeName]; ok {
		return exchange[tradeType]
	}
	return nil
}

// GetAllPriceFeeds 获取所有已初始化的PriceFeed
// 动态遍历所有交易所和交易类型的组合，返回有效的PriceFeed列表
func (m *WebSocketManager) GetAllPriceFeeds() []port.PriceFeed {
	var feeds []port.PriceFeed

	// 遍历所有交易所
	for _, exchClients := range m.clients {
		// 遍历该交易所的所有交易类型
		for _, wsClients := range exchClients {
			if wsClients != nil && wsClients.PriceFeed != nil {
				feeds = append(feeds, wsClients.PriceFeed)
			}
		}
	}

	return feeds
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
