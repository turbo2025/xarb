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
	// if binanceSpot := apiClients.ExchangeRegistry.BinanceSpot(); binanceSpot != nil {
	// 	factory.LogSpotBalance(ctx, factory.ExchangeBinance, binanceSpot.Account)
	// }
	// if bybitSpot := apiClients.ExchangeRegistry.BybitSpot(); bybitSpot != nil {
	// 	factory.LogSpotBalance(ctx, factory.ExchangeBybit, bybitSpot.Account)
	// }

	// Get Binance perpetual contract margin
	// if binancePerpetual := apiClients.ExchangeRegistry.BinancePerpetual(); binancePerpetual != nil {
	// 	if account, err := binancePerpetual.Account.GetAccount(ctx); err == nil {
	// 		log.Info().
	// 			Float64("total_margin", account.TotalMargin).
	// 			Float64("used_margin", account.UsedMargin).
	// 			Float64("avail_margin", account.AvailMargin).
	// 			Msgf("✓ Binance perpetual account")
	// 	} else {
	// 		log.Warn().Err(err).Msg("failed to fetch binance perpetual account info")
	// 	}
	// }

	// Get Bybit perpetual contract margin
	// if bybitPerpetual := apiClients.ExchangeRegistry.BybitPerpetual(); bybitPerpetual != nil {
	// 	if account, err := bybitPerpetual.Account.GetAccount(ctx); err == nil {
	// 		log.Info().
	// 			Float64("total_margin", account.TotalMargin).
	// 			Float64("used_margin", account.UsedMargin).
	// 			Float64("avail_margin", account.AvailMargin).
	// 			Msgf("✓ Bybit perpetual account")
	// 	} else {
	// 		log.Warn().Err(err).Msg("failed to fetch bybit perpetual account info")
	// 	}
	// }

	// 获取现货账户
	// okxSpot := apiClients.ExchangeRegistry.OKXSpot()
	// balance, err := okxSpot.Account.GetBalance(ctx)
	// if err != nil {
	// 	log.Warn().Err(err).Msg("failed to fetch okx spot balance")
	// } else {
	// 	log.Info().
	// 		Float64("spot_balance_usdt", balance).
	// 		Msgf("✓ OKX spot balance")
	// }

	// // Get perpetual margin
	// okxPerpetual := apiClients.ExchangeRegistry.OKXPerpetual()
	// account, err := okxPerpetual.Account.GetAccount(ctx)
	// // 返回：TotalMargin, AvailMargin, UsedMargin
	// if err != nil {
	// 	log.Warn().Err(err).Msg("failed to fetch okx perpetual account info")
	// } else {
	// 	log.Info().
	// 		Float64("total_margin", account.TotalMargin).
	// 		Float64("used_margin", account.UsedMargin).
	// 		Float64("avail_margin", account.AvailMargin).
	// 		Msgf("✓ OKX perpetual account")
	// }
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
		if err := registry.Register(exchangeName, &exchCfg); err != nil {
			return fmt.Errorf("failed to register %s: %w", exchangeName, err)
		}

		log.Info().Msgf("✓ %s clients registered", exchangeName)
	}

	return nil
}
