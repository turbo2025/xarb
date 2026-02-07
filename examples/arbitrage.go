package examples

import (
	"context"
	"log"
	"time"

	"xarb/internal/application/service"
	"xarb/internal/domain/model"
	"xarb/internal/infrastructure/storage/sqlite"
)

// DemonstrateArbitrage 套利系统完整演示
func DemonstrateArbitrage(repo *sqlite.Repo) {
	arbRepo := sqlite.NewArbitrageRepo(repo.GetDB())
	calc := service.NewArbitrageCalculator(0.0002)
	arbService := service.NewArbitrageService(arbRepo, calc, 0.01, 0.0002)

	ctx := context.Background()

	log.Println("\n=== 价差套利示例 ===")
	demonstrateSpreadArbitrage(ctx, arbService, arbRepo)

	log.Println("\n=== 资金费率套利示例 ===")
	demonstrateFundingArbitrage(ctx, arbService, arbRepo)

	log.Println("\n=== 持仓管理示例 ===")
	demonstratePositionManagement(ctx, arbService, arbRepo)

	log.Println("\n✓ 所有示例完成")
}

// 展示价差套利
func demonstrateSpreadArbitrage(ctx context.Context, svc *service.ArbitrageServiceImpl, repo *sqlite.ArbitrageRepo) {
	// Simulate prices fetched from exchanges
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
		Price:     42900, // 100 cheaper
		Funding:   0.00008,
		NextTime:  time.Now().Add(1 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	// Save prices
	_ = repo.SavePerpetualPrice(ctx, binancePrice)
	_ = repo.SavePerpetualPrice(ctx, bybitPrice)

	// Scan opportunities
	if err := svc.ScanSpreadOpportunities(ctx, binancePrice, bybitPrice); err != nil {
		log.Printf("错误: %v", err)
		return
	}

	// 获取并显示机会
	spread, err := repo.GetLatestSpreadBySymbol(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("错误: %v", err)
		return
	}

	if spread != nil {
		log.Printf("✓ 价差机会:")
		log.Printf("  - 交易对: %s", spread.Symbol)
		log.Printf("  - 做多: %s @ %.2f USDT", spread.LongExchange, spread.LongPrice)
		log.Printf("  - 做空: %s @ %.2f USDT", spread.ShortExchange, spread.ShortPrice)
		log.Printf("  - 价差: %.4f%%", spread.Spread)
		log.Printf("  - 预期利润: %.4f%%", spread.ProfitPercent)
	}
}

// demonstrateFundingArbitrage 展示资金费率套利
func demonstrateFundingArbitrage(ctx context.Context, svc *service.ArbitrageServiceImpl, repo *sqlite.ArbitrageRepo) {
	// Simulate prices with different funding rates
	binancePrice := &model.PerpetualPrice{
		Exchange:  "binance",
		Symbol:    "ETHUSDT",
		Price:     2300,
		Funding:   0.0003, // High funding rate
		NextTime:  time.Now().Add(8 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	bybitPrice := &model.PerpetualPrice{
		Exchange:  "bybit",
		Symbol:    "ETHUSDT",
		Price:     2300,
		Funding:   0.0001, // Low funding rate
		NextTime:  time.Now().Add(1 * time.Hour),
		Timestamp: time.Now().UnixMilli(),
	}

	// Save prices
	_ = repo.SavePerpetualPrice(ctx, binancePrice)
	_ = repo.SavePerpetualPrice(ctx, bybitPrice)

	// Scan 24-hour funding arbitrage opportunities
	if err := svc.ScanFundingOpportunities(ctx, binancePrice, bybitPrice, 24); err != nil {
		log.Printf("错误: %v", err)
		return
	}

	// 获取并显示机会
	funding, err := repo.GetLatestFundingBySymbol(ctx, "ETHUSDT")
	if err != nil {
		log.Printf("错误: %v", err)
		return
	}

	if funding != nil {
		log.Printf("✓ 资金费机会:")
		log.Printf("  - 交易对: %s", funding.Symbol)
		log.Printf("  - %s 资金费: %.5f", funding.LongExchange, funding.LongFunding)
		log.Printf("  - %s 资金费: %.5f", funding.ShortExchange, funding.ShortFunding)
		log.Printf("  - 资金费差: %.5f", funding.FundingDiff)
		log.Printf("  - 24小时预期回报: %.4f%%", funding.ExpectedReturn)
	}
}

// 展示持仓管理
func demonstratePositionManagement(ctx context.Context, svc *service.ArbitrageServiceImpl, repo *sqlite.ArbitrageRepo) {
	// 开仓
	err := svc.OpenPosition(ctx, "BTCUSDT", "binance", "bybit", 0.1, 43000, 42900)
	if err != nil {
		log.Printf("错误开仓: %v", err)
		return
	}
	log.Printf("✓ 已开仓")

	// 查询持仓
	positions, err := repo.ListOpenPositions(ctx)
	if err != nil {
		log.Printf("错误查询: %v", err)
		return
	}

	log.Printf("✓ 开仓数: %d", len(positions))
	for _, pos := range positions {
		log.Printf("  - ID: %s", pos.ID)
		log.Printf("  - 交易对: %s", pos.Symbol)
		log.Printf("  - 数量: %.4f", pos.Quantity)
		log.Printf("  - 状态: %s", pos.Status)
		log.Printf("  - 入场点差: %.4f%%", pos.EntrySpread)
	}
}
