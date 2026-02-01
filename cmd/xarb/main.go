package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/logger"
	"xarb/internal/interfaces/console"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Setup()

	configPath := flag.String("config", "configs/config.toml", "path to config.toml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config", *configPath).Msg("load config failed")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// output sink (console)
	sink := console.NewSink()

	// feeds (infrastructure -> application ports)
	var feeds []monitor.PriceFeed
	if cfg.Exchange.Binance.Enabled {
		feeds = append(feeds, binance.NewFuturesMiniTickerFeed(cfg.Exchange.Binance.WsURL))
	} else {
		log.Warn().Msg("binance disabled by config")
	}
	if cfg.Exchange.Bybit.Enabled {
		feeds = append(feeds, bybit.NewLinearTickerFeed(cfg.Exchange.Bybit.WsURL))
	} else {
		log.Warn().Msg("bybit disabled by config")
	}

	if len(feeds) == 0 {
		log.Fatal().Msg("no exchange feeds enabled")
	}

	// monitor usecase
	svc := monitor.NewService(monitor.ServiceDeps{
		Feeds:          feeds,
		Symbols:        cfg.Symbols.List,
		PrintEveryMin:  cfg.App.PrintEveryMin,
		DeltaThreshold: cfg.Arbitrage.DeltaThreshold,
		Sink:           sink,
		Repo:           monitor.NewNoopRepo(), // placeholder: later replace with redis/pg/sqlite repo
	})

	log.Info().
		Str("config", *configPath).
		Int("symbols", len(cfg.Symbols.List)).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Msg("xarb started")

	if err := svc.Run(ctx); err != nil {
		log.Error().Err(err).Msg("monitor service exited")
	}
}
