package application

import (
	"context"
	"fmt"
	"time"

	"binance-ws/domain"
	"binance-ws/infrastructure/exchange"
	"binance-ws/infrastructure/storage"
)

// PriceUpdateService handles price updates from exchanges
type PriceUpdateService struct {
	board   *domain.Board
	storage storage.Storage
	logger  Logger
}

// Logger interface for application layer
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewPriceUpdateService creates a new service
func NewPriceUpdateService(board *domain.Board, storage storage.Storage, logger Logger) *PriceUpdateService {
	return &PriceUpdateService{
		board:   board,
		storage: storage,
		logger:  logger,
	}
}

// HandlePriceUpdate processes a price update from an exchange
func (s *PriceUpdateService) HandlePriceUpdate(exchange, symbol, price string) error {
	if s.board.Update(exchange, symbol, price) {
		// Store the price update
		snapshot := &storage.PriceSnapshot{
			ID:        fmt.Sprintf("%s_%s_%d", exchange, symbol, time.Now().UnixNano()),
			Exchange:  exchange,
			Symbol:    symbol,
			Price:     0, // Parse if needed
			Timestamp: time.Now(),
		}

		// Attempt to store, but don't fail if storage is unavailable
		_ = s.storage.Prices().SavePrice(context.Background(), snapshot)
	}
	return nil
}

// SnapshotPrinterService prints periodic price snapshots
type SnapshotPrinterService struct {
	board    *domain.Board
	renderer Renderer
	logger   Logger
}

// Renderer interface for presentation layer
type Renderer interface {
	RenderLine(symbols []string, snapshot map[string]*domain.SymbolState, live bool) string
}

// NewSnapshotPrinterService creates a new service
func NewSnapshotPrinterService(board *domain.Board, renderer Renderer, logger Logger) *SnapshotPrinterService {
	return &SnapshotPrinterService{
		board:    board,
		renderer: renderer,
		logger:   logger,
	}
}

// PrintSnapshot prints the current state as a snapshot
func (s *SnapshotPrinterService) PrintSnapshot() {
	symbols := s.board.GetSymbols()
	snapshot := s.board.GetSnapshot()

	line := s.renderer.RenderLine(symbols, snapshot, false)
	fmt.Printf("\n%s %s\n", time.Now().Format("2006-01-02 15:04:05"), line)

	// Also print live line
	liveLine := s.renderer.RenderLine(symbols, snapshot, true)
	fmt.Print(liveLine)
}

// ExchangeRunnerService manages exchange connections
type ExchangeRunnerService struct {
	updateService  *PriceUpdateService
	exchangeLogger ExchangeLogger
	logger         Logger
}

// ExchangeLogger implements exchange.Logger interface
type ExchangeLogger struct {
	logger Logger
}

func (el *ExchangeLogger) Error(msg string, err error) {
	el.logger.Errorf("%s: %v", msg, err)
}

func (el *ExchangeLogger) Warn(msg string) {
	el.logger.Warnf(msg)
}

func (el *ExchangeLogger) Info(msg string) {
	el.logger.Infof(msg)
}

func (el *ExchangeLogger) Debug(msg string) {
	el.logger.Infof("[DEBUG] %s", msg)
}

// NewExchangeRunnerService creates a new service
func NewExchangeRunnerService(updateService *PriceUpdateService, logger Logger) *ExchangeRunnerService {
	return &ExchangeRunnerService{
		updateService:  updateService,
		exchangeLogger: ExchangeLogger{logger: logger},
		logger:         logger,
	}
}

// RunExchange starts a exchange runner
func (s *ExchangeRunnerService) RunExchange(ctx context.Context, exch exchange.Exchange) {
	runner := &exchange.Runner{
		Exchange: exch,
		Logger:   &s.exchangeLogger,
	}

	runner.Run(ctx, func(exchangeName, symbol, price string) error {
		return s.updateService.HandlePriceUpdate(exchangeName, symbol, price)
	})
}
