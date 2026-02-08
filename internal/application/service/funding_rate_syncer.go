package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"

	"github.com/rs/zerolog/log"
)

// FundingRateSyncer 资金费率同步器
type FundingRateSyncer struct {
	binanceClient *binance.FundingRateClient
	bybitClient   *bybit.FundingRateClient
	arbRepo       port.ArbitrageRepository
	interval      time.Duration
}

// NewFundingRateSyncer 创建资金费率同步器
func NewFundingRateSyncer(
	binanceRestURL string,
	bybitRestURL string,
	arbRepo port.ArbitrageRepository,
	interval time.Duration,
) *FundingRateSyncer {
	if interval <= 0 {
		interval = 1 * time.Hour // 默认1小时同步一次
	}
	return &FundingRateSyncer{
		binanceClient: binance.NewFundingRateClient(binanceRestURL),
		bybitClient:   bybit.NewFundingRateClient(bybitRestURL),
		arbRepo:       arbRepo,
		interval:      interval,
	}
}

// Start 启动后台同步任务
func (s *FundingRateSyncer) Start(ctx context.Context, symbols []string) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 首次立即同步
	s.syncFundingRates(ctx, symbols)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.syncFundingRates(ctx, symbols)
			}
		}
	}()

	return nil
}

// syncFundingRates 同步资金费率
func (s *FundingRateSyncer) syncFundingRates(ctx context.Context, symbols []string) {
	log.Debug().Msg("syncing funding rates")

	// 从 Binance 获取资金费率
	for _, symbol := range symbols {
		// 构建 Binance 符号格式（BTCUSDT）
		binanceSymbol := symbol
		if len(symbol) > 0 && symbol[len(symbol)-1] != 'T' {
			// 可能需要符号映射，这里暂时跳过
			continue
		}

		rate, err := s.binanceClient.GetFundingRate(binanceSymbol)
		if err != nil {
			log.Warn().
				Str("exchange", "BINANCE").
				Str("symbol", binanceSymbol).
				Err(err).
				Msg("failed to get funding rate")
			continue
		}

		if rate != nil {
			rateNum, _ := strconv.ParseFloat(rate.FundingRate, 64)
			log.Info().
				Str("exchange", "BINANCE").
				Str("symbol", binanceSymbol).
				Float64("funding_rate", rateNum).
				Msg("synced funding rate")

			// 保存到缓存或数据库（可选）
			_ = s.arbRepo
		}
	}

	// 从 Bybit 获取资金费率
	fundingRates, err := s.bybitClient.GetFundingRateForSymbols(symbols)
	if err == nil {
		for symbol, rate := range fundingRates {
			rateNum, _ := strconv.ParseFloat(rate, 64)
			log.Info().
				Str("exchange", "BYBIT").
				Str("symbol", symbol).
				Float64("funding_rate", rateNum).
				Msg("synced funding rate")
		}
	}
}

// SyncSingleSymbol 同步单个符号的资金费率
func (s *FundingRateSyncer) SyncSingleSymbol(ctx context.Context, exchange, symbol string) (*FundingRate, error) {
	switch exchange {
	case "BINANCE":
		rate, err := s.binanceClient.GetFundingRate(symbol)
		if err != nil {
			return nil, fmt.Errorf("binance error: %w", err)
		}
		if rate == nil {
			return nil, fmt.Errorf("no funding rate returned")
		}
		rateNum, _ := strconv.ParseFloat(rate.FundingRate, 64)
		return &FundingRate{
			Exchange: exchange,
			Symbol:   symbol,
			Rate:     rateNum,
			Time:     time.UnixMilli(rate.FundingTime),
		}, nil

	case "BYBIT":
		rate, err := s.bybitClient.GetFundingRate(symbol)
		if err != nil {
			return nil, fmt.Errorf("bybit error: %w", err)
		}
		rateNum, _ := strconv.ParseFloat(rate, 64)
		return &FundingRate{
			Exchange: exchange,
			Symbol:   symbol,
			Rate:     rateNum,
			Time:     time.Now(),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchange)
	}
}

// FundingRate 资金费率数据
type FundingRate struct {
	Exchange string
	Symbol   string
	Rate     float64
	Time     time.Time
}
