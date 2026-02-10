package factory

import (
	"fmt"

	"xarb/internal/infrastructure/config"

	"github.com/rs/zerolog/log"
)

// APIClients API 客户端容器
// 职责: 只管理交易所客户端的初始化和注册
type APIClients struct {
	ExchangeRegistry *ExchangeClientRegistry
}

// NewAPIClients 初始化所有交易所客户端
// 策略: 动态遍历 cfg.Exchanges 并注册所有启用的交易所
func NewAPIClients(cfg *config.Config) (*APIClients, error) {
	// 验证配置
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	registry := NewExchangeClientRegistry()

	// 动态注册已启用的交易所
	if err := registerExchanges(registry, cfg); err != nil {
		return nil, err
	}

	return &APIClients{
		ExchangeRegistry: registry,
	}, nil
}

// registerExchanges 遍历所有启用的交易所并注册
// 直接从 cfg.Exchanges map 中读取，完全动态化
func registerExchanges(registry *ExchangeClientRegistry, cfg *config.Config) error {
	for exchangeName, exchCfg := range cfg.Exchanges {
		// 跳过未启用的交易所
		if !exchCfg.Enabled {
			continue
		}

		// 调用通用注册方法
		if err := registry.Register(exchangeName, exchCfg.APIKey, exchCfg.SecretKey, exchCfg.FuturesURL, exchCfg.SpotURL); err != nil {
			return fmt.Errorf("failed to register %s: %w", exchangeName, err)
		}

		log.Info().Msgf("✓ %s clients registered", exchangeName)
	}

	return nil
}
