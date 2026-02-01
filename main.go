package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"binance-ws/application"
	"binance-ws/domain"
	"binance-ws/infrastructure/exchange"
	"binance-ws/infrastructure/storage"
	"binance-ws/presentation"
)

// ZerologAdapter adapts zerolog to application.Logger
type ZerologAdapter struct {
}

func (za *ZerologAdapter) Infof(format string, args ...interface{}) {
	log.Info().Msgf(format, args...)
}

func (za *ZerologAdapter) Warnf(format string, args ...interface{}) {
	log.Warn().Msgf(format, args...)
}

func (za *ZerologAdapter) Errorf(format string, args ...interface{}) {
	log.Error().Msgf(format, args...)
}

// SnapshotPrinterRenderer adapts presentation.Renderer to application.Renderer
type SnapshotPrinterRenderer struct {
	renderer *presentation.Renderer
}

func (spr *SnapshotPrinterRenderer) RenderLine(symbols []string, snapshot map[string]*domain.SymbolState, live bool) string {
	return spr.renderer.RenderLine(symbols, snapshot, live)
}

func setupLogger() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func main() {
	setupLogger()

	configPath := flag.String("config", "config.toml", "path to config.toml")
	flag.Parse()

	// Load configuration
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config", *configPath).Msg("load config failed")
	}

	symbols := cfg.Symbols.List

	// Initialize domain layer
	board := domain.NewBoard(symbols, cfg.Arbitrage.DeltaThreshold)

	// Initialize infrastructure layer
	storage := storage.NewInMemoryStorage()
	defer storage.Close()

	// Initialize presentation layer
	renderer := presentation.NewRenderer(cfg.Arbitrage.DeltaThreshold)
	renderAdapter := &SnapshotPrinterRenderer{renderer: renderer}

	// Initialize application layer
	loggerAdapter := &ZerologAdapter{}
	updateService := application.NewPriceUpdateService(board, storage, loggerAdapter)
	snapshotService := application.NewSnapshotPrinterService(board, renderAdapter, loggerAdapter)
	exchangeService := application.NewExchangeRunnerService(updateService, loggerAdapter)

	fmt.Print("\n")
	log.Info().
		Str("config", *configPath).
		Int("print_every_min", cfg.App.PrintEveryMin).
		Float64("delta_threshold", cfg.Arbitrage.DeltaThreshold).
		Bool("binance_enabled", cfg.Exchange.Binance.Enabled).
		Bool("bybit_enabled", cfg.Exchange.Bybit.Enabled).
		Int("symbols", len(symbols)).
		Msg("started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start exchange runners
	if cfg.Exchange.Binance.Enabled {
		binanceExch := exchange.NewBinance(cfg.Exchange.Binance.WsURL, symbols)
		go exchangeService.RunExchange(ctx, binanceExch)
	} else {
		fmt.Print("\n")
		log.Warn().Msg("binance disabled by config")
	}

	if cfg.Exchange.Bybit.Enabled {
		bybitExch := exchange.NewBybit(cfg.Exchange.Bybit.WsURL, symbols)
		go exchangeService.RunExchange(ctx, bybitExch)
	} else {
		fmt.Print("\n")
		log.Warn().Msg("bybit disabled by config")
	}

	// Start snapshot printer
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.App.PrintEveryMin) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshotService.PrintSnapshot()
			}
		}
	}()

	// Print live updates
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				symbols := board.GetSymbols()
				snapshot := board.GetSnapshot()
				line := renderer.RenderLine(symbols, snapshot, true)
				fmt.Print(line)
			}
		}
	}()

	<-ctx.Done()
	fmt.Print("\n")
	log.Warn().Msg("exit")
}
