package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

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

	serviceCtx, err := svc.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("service context initialization failed")
		return
	}
	defer serviceCtx.Close()
	// 运行所有服务
	if err := serviceCtx.Run(ctx); err != nil {
		log.Error().Err(err).Msg("service exited with error")
		return
	}

	log.Info().
		Str("config", *configPath).
		Int("coins", len(cfg.Symbols.Coins)).
		Str("quote", cfg.Symbols.Quote).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("sqlite_enabled", cfg.SQLite.Enabled).
		Bool("redis_enabled", cfg.Redis.Enabled).
		Msg("xarb started")
}
