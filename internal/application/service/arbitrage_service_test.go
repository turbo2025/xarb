package service

import (
	"context"
	"testing"
	"time"

	"xarb/internal/domain/model"
)

type MockArbitrageRepo struct {
	spreads  map[string]*model.SpreadArbitrage
	fundings map[string]*model.FundingArbitrage
	prices   map[string]*model.PerpetualPrice
}

func NewMockArbitrageRepo() *MockArbitrageRepo {
	return &MockArbitrageRepo{
		spreads:  make(map[string]*model.SpreadArbitrage),
		fundings: make(map[string]*model.FundingArbitrage),
		prices:   make(map[string]*model.PerpetualPrice),
	}
}

func (m *MockArbitrageRepo) SaveSpreadOpportunity(ctx context.Context, arb *model.SpreadArbitrage) error {
	m.spreads[arb.Symbol] = arb
	return nil
}

func (m *MockArbitrageRepo) GetLatestSpreadBySymbol(ctx context.Context, symbol string) (*model.SpreadArbitrage, error) {
	return m.spreads[symbol], nil
}

func (m *MockArbitrageRepo) SaveFundingOpportunity(ctx context.Context, arb *model.FundingArbitrage) error {
	m.fundings[arb.Symbol] = arb
	return nil
}

func (m *MockArbitrageRepo) GetLatestFundingBySymbol(ctx context.Context, symbol string) (*model.FundingArbitrage, error) {
	return m.fundings[symbol], nil
}

func (m *MockArbitrageRepo) CreatePosition(ctx context.Context, pos *model.ArbitragePosition) error {
	return nil
}

func (m *MockArbitrageRepo) UpdatePosition(ctx context.Context, pos *model.ArbitragePosition) error {
	return nil
}

func (m *MockArbitrageRepo) GetPosition(ctx context.Context, id string) (*model.ArbitragePosition, error) {
	return nil, nil
}

func (m *MockArbitrageRepo) ListOpenPositions(ctx context.Context) ([]*model.ArbitragePosition, error) {
	return nil, nil
}

func (m *MockArbitrageRepo) SavePerpetualPrice(ctx context.Context, price *model.PerpetualPrice) error {
	m.prices[price.Exchange+"_"+price.Symbol] = price
	return nil
}

func (m *MockArbitrageRepo) GetLatestPrice(ctx context.Context, exchange, symbol string) (*model.PerpetualPrice, error) {
	return m.prices[exchange+"_"+symbol], nil
}

// TestSpreadArbitrage 测试价差套利计算
func TestSpreadArbitrage(t *testing.T) {
	calc := NewArbitrageCalculator(0.0002)
	repo := NewMockArbitrageRepo()
	svc := NewArbitrageService(repo, calc, 0.01, 0.0002)

	binancePrice := &model.PerpetualPrice{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Price:     43000,
		Funding:   0.0001,
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	bybitPrice := &model.PerpetualPrice{
		Exchange:  "bybit",
		Symbol:    "BTCUSDT",
		Price:     42900, // 100 USDT 便宜
		Funding:   0.00008,
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	ctx := context.Background()
	err := svc.ScanSpreadOpportunities(ctx, binancePrice, bybitPrice)
	if err != nil {
		t.Fatalf("scan spread opportunities failed: %v", err)
	}

	spread, err := repo.GetLatestSpreadBySymbol(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("get spread failed: %v", err)
	}

	if spread == nil {
		t.Fatal("spread should not be nil")
	}

	// 检查价差是否大于最小值
	if spread.ProfitPercent <= 0 {
		t.Errorf("profit percent should be > 0, got %f", spread.ProfitPercent)
	}

	t.Logf("Spread: %f%%, Profit: %f%%", spread.Spread, spread.ProfitPercent)
}

// TestFundingArbitrage 测试资金费率套利
func TestFundingArbitrage(t *testing.T) {
	calc := NewArbitrageCalculator(0.0002)
	repo := NewMockArbitrageRepo()
	svc := NewArbitrageService(repo, calc, 0.01, 0.0002)

	binancePrice := &model.PerpetualPrice{
		Exchange:  "binance",
		Symbol:    "ETHUSDT",
		Price:     2300,
		Funding:   0.0003, // 高资金费
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	bybitPrice := &model.PerpetualPrice{
		Exchange:  "bybit",
		Symbol:    "ETHUSDT",
		Price:     2300,
		Funding:   0.0001, // 低资金费
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	ctx := context.Background()
	err := svc.ScanFundingOpportunities(ctx, binancePrice, bybitPrice, 24)
	if err != nil {
		t.Fatalf("scan funding opportunities failed: %v", err)
	}

	funding, err := repo.GetLatestFundingBySymbol(ctx, "ETHUSDT")
	if err != nil {
		t.Fatalf("get funding failed: %v", err)
	}

	if funding == nil {
		t.Fatal("funding should not be nil")
	}

	if funding.ExpectedReturn <= 0 {
		t.Errorf("expected return should be > 0, got %f", funding.ExpectedReturn)
	}

	t.Logf("Funding Diff: %f, Expected Return: %f%%", funding.FundingDiff, funding.ExpectedReturn)
}

// TestCalculatorAccuracy 测试计算器精度
func TestCalculatorAccuracy(t *testing.T) {
	calc := NewArbitrageCalculator(0.0002)

	longPrice := &model.PerpetualPrice{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Price:     43000,
		Funding:   0.0001,
		NextTime:  time.Now(),
		Timestamp: time.Now().UnixMilli(),
	}

	shortPrice := &model.PerpetualPrice{
		Exchange:  "bybit",
		Symbol:    "BTCUSDT",
		Price:     42900,
		Funding:   0.00008,
		NextTime:  time.Now(),
		Timestamp: time.Now().UnixMilli(),
	}

	spread := calc.CalculateSpread(longPrice, shortPrice, 0.0002)
	if spread == nil {
		t.Fatal("spread should not be nil")
	}

	// 手续费成本：43000 * 0.0002 + 42900 * 0.0002 = 8.6 + 8.58 = 17.18
	// 价差：43000 - 42900 = 100
	// 价差率：100 / 42900 = 0.233%
	expectedSpread := (100.0 / 42900) * 100
	expectedFee := (43000*0.0002 + 42900*0.0002) / 42900 * 100

	if spread.Spread < expectedSpread-0.01 || spread.Spread > expectedSpread+0.01 {
		t.Errorf("spread mismatch: expected ~%.4f, got %.4f", expectedSpread, spread.Spread)
	}

	t.Logf("Spread: %.4f%%, Fee Cost: %.4f%%, Profit: %.4f%%", spread.Spread, expectedFee, spread.ProfitPercent)
}
