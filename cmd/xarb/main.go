package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/logger"
	"xarb/internal/infrastructure/svc"

	"github.com/rs/zerolog/log"
)

func main() {
	// Setup context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Setup()
	// Parse flags
	configPath := flag.String("config", "configs/config.toml", "path to config.toml")
	flag.Parse()
	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config", *configPath).Msg("load config failed")
		return
	}

	serviceCtx, err := svc.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("service context initialization failed")
		return
	}
	defer serviceCtx.Close()
	log.Info().
		Str("config", *configPath).
		Int("symbols", len(cfg.Symbols.List)).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("storage_enabled", cfg.Storage.Enabled).
		Msg("xarb started")

	service := monitor.NewService(serviceCtx.BuildMonitorServiceDeps())
	if err := service.Run(ctx); err != nil {
		log.Error().Err(err).Msg("monitor service exited")
	}

}
