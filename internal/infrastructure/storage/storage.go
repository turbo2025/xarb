package storage

import (
	"context"
	"time"
)

// PriceSnapshot represents a single price record
type PriceSnapshot struct {
	ID        string
	Exchange  string
	Symbol    string
	Price     float64
	Timestamp time.Time
}

// ArbitrageOpportunity represents a detected arbitrage
type ArbitrageOpportunity struct {
	ID           string
	Symbol       string
	BinancePrice float64
	BybitPrice   float64
	Delta        float64
	Timestamp    time.Time
}

// PriceRepository defines operations for storing prices
type PriceRepository interface {
	// SavePrice stores a single price snapshot
	SavePrice(ctx context.Context, snapshot *PriceSnapshot) error

	// SavePrices stores multiple price snapshots
	SavePrices(ctx context.Context, snapshots []*PriceSnapshot) error

	// GetPrices retrieves prices for a symbol in time range
	GetPrices(ctx context.Context, symbol string, start, end time.Time) ([]*PriceSnapshot, error)

	// DeleteOldPrices removes prices older than the specified time
	DeleteOldPrices(ctx context.Context, before time.Time) error
}

// ArbitrageRepository defines operations for storing arbitrage opportunities
type ArbitrageRepository interface {
	// SaveOpportunity stores an arbitrage opportunity
	SaveOpportunity(ctx context.Context, opp *ArbitrageOpportunity) error

	// GetOpportunities retrieves opportunities in time range
	GetOpportunities(ctx context.Context, start, end time.Time) ([]*ArbitrageOpportunity, error)

	// DeleteOldOpportunities removes opportunities older than the specified time
	DeleteOldOpportunities(ctx context.Context, before time.Time) error
}

// Storage defines the overall storage interface
type Storage interface {
	Prices() PriceRepository
	Arbitrage() ArbitrageRepository
	Close() error
}

// InMemoryPriceRepository is a simple in-memory implementation
type InMemoryPriceRepository struct {
	prices []*PriceSnapshot
}

// NewInMemoryPriceRepository creates a new in-memory repository
func NewInMemoryPriceRepository() *InMemoryPriceRepository {
	return &InMemoryPriceRepository{
		prices: make([]*PriceSnapshot, 0),
	}
}

func (r *InMemoryPriceRepository) SavePrice(ctx context.Context, snapshot *PriceSnapshot) error {
	r.prices = append(r.prices, snapshot)
	return nil
}

func (r *InMemoryPriceRepository) SavePrices(ctx context.Context, snapshots []*PriceSnapshot) error {
	r.prices = append(r.prices, snapshots...)
	return nil
}

func (r *InMemoryPriceRepository) GetPrices(ctx context.Context, symbol string, start, end time.Time) ([]*PriceSnapshot, error) {
	var result []*PriceSnapshot
	for _, p := range r.prices {
		if p.Symbol == symbol && p.Timestamp.After(start) && p.Timestamp.Before(end) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *InMemoryPriceRepository) DeleteOldPrices(ctx context.Context, before time.Time) error {
	filtered := make([]*PriceSnapshot, 0)
	for _, p := range r.prices {
		if p.Timestamp.After(before) {
			filtered = append(filtered, p)
		}
	}
	r.prices = filtered
	return nil
}

// InMemoryArbitrageRepository is a simple in-memory implementation
type InMemoryArbitrageRepository struct {
	opportunities []*ArbitrageOpportunity
}

// NewInMemoryArbitrageRepository creates a new in-memory repository
func NewInMemoryArbitrageRepository() *InMemoryArbitrageRepository {
	return &InMemoryArbitrageRepository{
		opportunities: make([]*ArbitrageOpportunity, 0),
	}
}

func (r *InMemoryArbitrageRepository) SaveOpportunity(ctx context.Context, opp *ArbitrageOpportunity) error {
	r.opportunities = append(r.opportunities, opp)
	return nil
}

func (r *InMemoryArbitrageRepository) GetOpportunities(ctx context.Context, start, end time.Time) ([]*ArbitrageOpportunity, error) {
	var result []*ArbitrageOpportunity
	for _, o := range r.opportunities {
		if o.Timestamp.After(start) && o.Timestamp.Before(end) {
			result = append(result, o)
		}
	}
	return result, nil
}

func (r *InMemoryArbitrageRepository) DeleteOldOpportunities(ctx context.Context, before time.Time) error {
	filtered := make([]*ArbitrageOpportunity, 0)
	for _, o := range r.opportunities {
		if o.Timestamp.After(before) {
			filtered = append(filtered, o)
		}
	}
	r.opportunities = filtered
	return nil
}

// InMemoryStorage implements Storage interface
type InMemoryStorage struct {
	prices    *InMemoryPriceRepository
	arbitrage *InMemoryArbitrageRepository
}

// NewInMemoryStorage creates a new in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		prices:    NewInMemoryPriceRepository(),
		arbitrage: NewInMemoryArbitrageRepository(),
	}
}

func (s *InMemoryStorage) Prices() PriceRepository {
	return s.prices
}

func (s *InMemoryStorage) Arbitrage() ArbitrageRepository {
	return s.arbitrage
}

func (s *InMemoryStorage) Close() error {
	return nil
}
