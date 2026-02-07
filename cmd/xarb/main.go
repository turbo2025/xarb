package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/container"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/logger"
	"xarb/internal/interfaces/console"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Setup()

	// Parse flags
	configPath := flag.String("config", "configs/config.toml", "path to config.toml")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config", *configPath).Msg("load config failed")
	}

	// Initialize container with all dependencies
	cont, err := container.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("container initialization failed")
	}
	defer cont.Close()

	// Setup context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize components
	sink := console.NewSink()
	feeds := initializeFeeds(cfg)
	if len(feeds) == 0 {
		log.Fatal().Msg("no exchange feeds enabled")
	}

	repo := monitor.NewNoopRepo()

	// Create service
	svc := monitor.NewService(monitor.ServiceDeps{
		Feeds:          feeds,
		Symbols:        cfg.Symbols.List,
		PrintEveryMin:  cfg.App.PrintEveryMin,
		DeltaThreshold: cfg.Arbitrage.DeltaThreshold,
		Sink:           sink,
		Repo:           repo,
	})

	// Log startup info
	log.Info().
		Str("config", *configPath).
		Int("symbols", len(cfg.Symbols.List)).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("storage_enabled", cfg.Storage.Enabled).
		Msg("xarb started")

	// Run service
	if err := svc.Run(ctx); err != nil {
		log.Error().Err(err).Msg("monitor service exited")
	}
}

// initializeFeeds 初始化交易所数据源
func initializeFeeds(cfg *config.Config) []monitor.PriceFeed {
	var feeds []monitor.PriceFeed

	if cfg.Exchange.Binance.Enabled {
		feeds = append(feeds, binance.NewFuturesMiniTickerFeed(cfg.Exchange.Binance.WsURL))
		log.Info().Msg("binance feed initialized")
	} else {
		log.Warn().Msg("binance disabled by config")
	}

	if cfg.Exchange.Bybit.Enabled {
		feeds = append(feeds, bybit.NewLinearTickerFeed(cfg.Exchange.Bybit.WsURL))
		log.Info().Msg("bybit feed initialized")
	} else {
		log.Warn().Msg("bybit disabled by config")
	}

	return feeds
}
