package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xarb/internal/application/usecase/monitor"
	"xarb/internal/infrastructure/config"
	"xarb/internal/infrastructure/exchange/binance"
	"xarb/internal/infrastructure/exchange/bybit"
	"xarb/internal/infrastructure/logger"

	// pgrepo "xarb/internal/infrastructure/storage/postgres"
	redisrepo "xarb/internal/infrastructure/storage/redis"
	sqliterepo "xarb/internal/infrastructure/storage/sqlite"
	"xarb/internal/interfaces/console"

	"github.com/redis/go-redis/v9"
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

	// sink (console)
	sink := console.NewSink()

	// feeds
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

	// repositories (composite)
	repo := monitor.NewNoopRepo()
	var repoList []monitor.RepositoryCloser

	if cfg.Storage.Enabled {
		var repos []monitor.Repository // local alias types in monitor package below

		// Redis
		if cfg.Storage.Redis.Enabled {
			rdb := redis.NewClient(&redis.Options{
				Addr:     cfg.Storage.Redis.Addr,
				Password: cfg.Storage.Redis.Password,
				DB:       cfg.Storage.Redis.DB,
			})
			ttl := time.Duration(cfg.Storage.Redis.TTLSeconds) * time.Second
			repos = append(repos, redisrepo.New(
				rdb,
				cfg.Storage.Redis.Prefix,
				ttl,
				cfg.Storage.Redis.SignalStream,
				cfg.Storage.Redis.SignalChannel,
			))
			log.Info().Str("addr", cfg.Storage.Redis.Addr).Msg("redis repo enabled")
		}

		// SQLite
		if cfg.Storage.SQLite.Enabled {
			r, err := sqliterepo.New(cfg.Storage.SQLite.Path)
			if err != nil {
				log.Fatal().Err(err).Msg("sqlite repo init failed")
			}
			repos = append(repos, r)
			repoList = append(repoList, r) // close on exit
			log.Info().Str("path", cfg.Storage.SQLite.Path).Msg("sqlite repo enabled")
		}

		// Postgres
		// if cfg.Storage.Postgres.Enabled {
		// 	r, err := pgrepo.New(cfg.Storage.Postgres.DSN)
		// 	if err != nil {
		// 		log.Fatal().Err(err).Msg("postgres repo init failed")
		// 	}
		// 	repos = append(repos, r)
		// 	repoList = append(repoList, r)
		// 	log.Info().Msg("postgres repo enabled")
		// }

		// if len(repos) > 0 {
		// 	repo = composite.New(repos...)
		// } else {
		// 	log.Warn().Msg("storage.enabled=true but no storage backend enabled")
		// }
	}

	// ensure closers closed
	defer func() {
		for _, c := range repoList {
			_ = c.Close()
		}
	}()

	svc := monitor.NewService(monitor.ServiceDeps{
		Feeds:          feeds,
		Symbols:        cfg.Symbols.List,
		PrintEveryMin:  cfg.App.PrintEveryMin,
		DeltaThreshold: cfg.Arbitrage.DeltaThreshold,
		Sink:           sink,
		Repo:           repo,
	})

	log.Info().
		Str("config", *configPath).
		Int("symbols", len(cfg.Symbols.List)).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("storage_enabled", cfg.Storage.Enabled).
		Msg("xarb started")

	if err := svc.Run(ctx); err != nil {
		log.Error().Err(err).Msg("monitor service exited")
	}
}
