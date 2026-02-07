package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"xarb/internal/domain/model"
)

func TestArbitrageRepoSpread(t *testing.T) {
	// 创建临时数据库
	tmpFile := "/tmp/test_arbitrage.db"
	defer os.Remove(tmpFile)

	// 初始化 repo
	repo, err := New(tmpFile)
	if err != nil {
		t.Fatalf("create repo failed: %v", err)
	}
	defer repo.Close()

	arbRepo := NewArbitrageRepo(repo.GetDB())
	ctx := context.Background()

	// 创建测试数据
	spread := &model.SpreadArbitrage{
		Symbol:        "BTCUSDT",
		LongExchange:  "bybit",
		ShortExchange: "binance",
		LongPrice:     42900,
		ShortPrice:    43000,
		Spread:        0.23,
		SpreadAbs:     100,
		ProfitPercent: 0.19,
		Timestamp:     time.Now().UnixMilli(),
	}

	// 保存
	if err := arbRepo.SaveSpreadOpportunity(ctx, spread); err != nil {
		t.Fatalf("save spread failed: %v", err)
	}

	// 查询
	retrieved, err := arbRepo.GetLatestSpreadBySymbol(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("get spread failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("spread should not be nil")
	}

	if retrieved.Symbol != "BTCUSDT" {
		t.Errorf("symbol mismatch: expected BTCUSDT, got %s", retrieved.Symbol)
	}

	if retrieved.ProfitPercent < 0.18 || retrieved.ProfitPercent > 0.20 {
		t.Errorf("profit percent mismatch: expected ~0.19, got %f", retrieved.ProfitPercent)
	}

	t.Logf("✓ Spread arbitrage saved and retrieved: %+v", retrieved)
}

func TestArbitrageRepoFunding(t *testing.T) {
	tmpFile := "/tmp/test_arbitrage2.db"
	defer os.Remove(tmpFile)

	repo, err := New(tmpFile)
	if err != nil {
		t.Fatalf("create repo failed: %v", err)
	}
	defer repo.Close()

	arbRepo := NewArbitrageRepo(repo.GetDB())
	ctx := context.Background()

	// 创建测试数据
	funding := &model.FundingArbitrage{
		Symbol:         "ETHUSDT",
		LongExchange:   "binance",
		ShortExchange:  "bybit",
		LongFunding:    0.0003,
		ShortFunding:   0.0001,
		FundingDiff:    0.0002,
		HoldingHours:   24,
		ExpectedReturn: 0.06,
		Timestamp:      time.Now().UnixMilli(),
	}

	// 保存
	if err := arbRepo.SaveFundingOpportunity(ctx, funding); err != nil {
		t.Fatalf("save funding failed: %v", err)
	}

	// 查询
	retrieved, err := arbRepo.GetLatestFundingBySymbol(ctx, "ETHUSDT")
	if err != nil {
		t.Fatalf("get funding failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("funding should not be nil")
	}

	if retrieved.FundingDiff < 0.00019 || retrieved.FundingDiff > 0.00021 {
		t.Errorf("funding diff mismatch: expected ~0.0002, got %f", retrieved.FundingDiff)
	}

	t.Logf("✓ Funding arbitrage saved and retrieved: %+v", retrieved)
}

func TestArbitrageRepoPositions(t *testing.T) {
	tmpFile := "/tmp/test_arbitrage3.db"
	defer os.Remove(tmpFile)

	repo, err := New(tmpFile)
	if err != nil {
		t.Fatalf("create repo failed: %v", err)
	}
	defer repo.Close()

	arbRepo := NewArbitrageRepo(repo.GetDB())
	ctx := context.Background()

	// 创建持仓
	pos := &model.ArbitragePosition{
		ID:              "pos_001",
		Symbol:          "BTCUSDT",
		LongExchange:    "bybit",
		ShortExchange:   "binance",
		Quantity:        0.1,
		LongEntryPrice:  42900,
		ShortEntryPrice: 43000,
		EntrySpread:     0.23,
		Status:          "open",
		OpenTime:        time.Now().UnixMilli(),
	}

	// 创建
	if err := arbRepo.CreatePosition(ctx, pos); err != nil {
		t.Fatalf("create position failed: %v", err)
	}

	// 查询
	retrieved, err := arbRepo.GetPosition(ctx, "pos_001")
	if err != nil {
		t.Fatalf("get position failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("position should not be nil")
	}

	if retrieved.Status != "open" {
		t.Errorf("status mismatch: expected open, got %s", retrieved.Status)
	}

	// 平仓
	pos.Status = "closed"
	pos.CloseTime = time.Now().UnixMilli()
	pos.RealizedPnL = 10.0
	if err := arbRepo.UpdatePosition(ctx, pos); err != nil {
		t.Fatalf("update position failed: %v", err)
	}

	// 查询开仓持仓（应该为空）
	openPositions, err := arbRepo.ListOpenPositions(ctx)
	if err != nil {
		t.Fatalf("list open positions failed: %v", err)
	}

	if len(openPositions) != 0 {
		t.Errorf("open positions should be empty, got %d", len(openPositions))
	}

	t.Logf("✓ Position lifecycle (create/update/close) completed successfully")
}

func TestArbitrageRepoPerpetualPrice(t *testing.T) {
	tmpFile := "/tmp/test_arbitrage4.db"
	defer os.Remove(tmpFile)

	repo, err := New(tmpFile)
	if err != nil {
		t.Fatalf("create repo failed: %v", err)
	}
	defer repo.Close()

	arbRepo := NewArbitrageRepo(repo.GetDB())
	ctx := context.Background()

	// 保存价格
	price := &model.PerpetualPrice{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Price:     43000,
		Funding:   0.0001,
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	if err := arbRepo.SavePerpetualPrice(ctx, price); err != nil {
		t.Fatalf("save price failed: %v", err)
	}

	// 查询
	retrieved, err := arbRepo.GetLatestPrice(ctx, "binance", "BTCUSDT")
	if err != nil {
		t.Fatalf("get price failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("price should not be nil")
	}

	if retrieved.Price != 43000 {
		t.Errorf("price mismatch: expected 43000, got %f", retrieved.Price)
	}

	if retrieved.Funding < 0.00009 || retrieved.Funding > 0.00011 {
		t.Errorf("funding mismatch: expected ~0.0001, got %f", retrieved.Funding)
	}

	t.Logf("✓ Perpetual price saved and retrieved: exchange=%s, symbol=%s, price=%.2f",
		retrieved.Exchange, retrieved.Symbol, retrieved.Price)
}
