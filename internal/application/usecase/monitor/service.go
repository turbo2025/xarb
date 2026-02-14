package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/application/service"
	domainservice "xarb/internal/domain/service"

	"github.com/rs/zerolog/log"
)

type PriceFeed = port.PriceFeed

type ServiceDeps struct {
	Feeds            []PriceFeed
	Symbols          []string
	PrintEveryMin    int
	DeltaThreshold   float64
	Sink             port.Sink
	Repo             port.Repository
	ArbitrageRepo    port.ArbitrageRepository         // 套利仓储
	ArbitrageCalc    *service.ArbitrageCalculator     // 套利计算器
	SymbolMapper     *domainservice.SymbolMapper      // 符号映射器（可选）
	OrderManager     *domainservice.OrderManager      // 订单执行器（期货用）
	Executor         *domainservice.ArbitrageExecutor // 套利分析器
	AccountManager   *domainservice.AccountManager    // 账户管理器（可选）
	TradeTypeManager *domainservice.TradeTypeManager  // 交易类型管理器（支持期货和现货）
}

type Service struct {
	deps     ServiceDeps
	st       *State
	fmt      *Formatter
	lastBand map[string]int  // -1/0/+1
	seenBand map[string]bool // 是否已建立基线

	// 用于订单执行的价格缓存
	pricesLock sync.RWMutex
	prices     map[string]map[string]float64 // symbol -> exchange -> price
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		deps:     deps,
		st:       NewState(deps.Symbols),
		fmt:      NewFormatter(deps.DeltaThreshold),
		lastBand: make(map[string]int, len(deps.Symbols)),
		seenBand: make(map[string]bool, len(deps.Symbols)),
		prices:   make(map[string]map[string]float64), // symbol -> exchange -> price
	}
}

func (s *Service) Run(ctx context.Context) error {
	if len(s.deps.Feeds) == 0 {
		return errors.New("no feeds")
	}

	merged := make(chan port.Tick, 1024)

	// start feeds
	for _, feed := range s.deps.Feeds {
		ch, err := feed.Subscribe(ctx, s.deps.Symbols)
		if err != nil {
			return err
		}
		go func(name string, in <-chan port.Tick) {
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-in:
					if !ok {
						return
					}
					merged <- t
				}
			}
		}(feed.Name(), ch)

		log.Info().Str("feed", feed.Name()).Msg("feed started")
	}

	// snapshot ticker
	snapTicker := time.NewTicker(time.Duration(s.deps.PrintEveryMin) * time.Minute)
	defer snapTicker.Stop()

	// initial live line
	_ = s.deps.Sink.WriteLive(s.fmt.Render(s.st, RenderLive))

	for {
		select {
		case <-ctx.Done():
			_ = s.deps.Sink.NewLine()
			return ctx.Err()

		case now := <-snapTicker.C:
			line := s.fmt.Render(s.st, RenderSnapshot)
			_ = s.deps.Sink.WriteSnapshot(now, line)
			// optional: persist snapshot
			if s.deps.Repo != nil {
				_ = s.deps.Repo.InsertSnapshot(ctx, now.UnixMilli(), line)
			}

		case t := <-merged:
			// 保存价格到缓存
			s.pricesLock.Lock()
			if s.prices[t.Symbol] == nil {
				s.prices[t.Symbol] = make(map[string]float64)
			}
			s.prices[t.Symbol][t.Exchange] = t.PriceNum
			s.pricesLock.Unlock()

			changed := s.st.Apply(t)
			if changed {
				line := s.fmt.Render(s.st, RenderLive)
				_ = s.deps.Sink.WriteLive(line)
			}

			// persist latest (optional)
			if s.deps.Repo != nil && t.PriceNum > 0 {
				_ = s.deps.Repo.UpsertLatestPrice(ctx, t.Exchange, t.Symbol, t.PriceNum, t.Ts)
			}

			// ---- threshold crossing detection ----
			delta, band, ok := s.st.DeltaBand(t.Symbol, s.deps.DeltaThreshold)
			if !ok {
				continue
			}

			prevBand, hasPrev := s.lastBand[t.Symbol]
			if !hasPrev {
				prevBand = 0
			}
			seen := s.seenBand[t.Symbol]

			// 建基线：第一次拿到有效 delta 不触发
			if !seen {
				s.lastBand[t.Symbol] = band
				s.seenBand[t.Symbol] = true
				continue
			}

			// 穿越阈值：band 变化且新 band != 0 才发信号
			if band != prevBand && band != 0 {
				payload := s.fmt.Render(s.st, RenderSnapshot) // 用快照格式（无 \r / 清行）
				// _ = s.deps.Repo.InsertSignal(ctx, time.Now().UnixMilli(), t.Symbol, delta, payload)
				// ⚠️ 信号直接打到 console（一次）
				s.deps.Sink.NewLine()
				log.Warn().
					Str("symbol", t.Symbol).
					Float64("delta", delta).
					Int("band", band).
					Float64("threshold", s.deps.DeltaThreshold).
					Msg(payload)

				// ✅ 新增：检测到套利机会，执行订单！
				if s.deps.OrderManager != nil && s.deps.Executor != nil {
					s.handleArbitrageSignal(ctx, t.Symbol, delta)
				}
			}

			// 更新 band（即使变回 0 也要更新，才能捕捉下一次穿越）
			s.lastBand[t.Symbol] = band
		}
	}
}

// handleArbitrageSignal 处理套利信号：执行订单并验证
func (s *Service) handleArbitrageSignal(ctx context.Context, symbol string, delta float64) {
	s.pricesLock.RLock()
	prices := s.prices[symbol]
	s.pricesLock.RUnlock()

	if len(prices) < 2 {
		log.Warn().Str("symbol", symbol).Msg("insufficient price data for arbitrage")
		return
	}

	// 获取 Binance 和 Bybit 的价格
	binancePrice, hasBinance := prices["Binance"]
	bybitPrice, hasBybit := prices["Bybit"]

	if !hasBinance || !hasBybit {
		log.Warn().
			Str("symbol", symbol).
			Bool("binance", hasBinance).
			Bool("bybit", hasBybit).
			Msg("missing exchange prices")
		return
	}

	log.Info().
		Str("symbol", symbol).
		Float64("binance_price", binancePrice).
		Float64("bybit_price", bybitPrice).
		Float64("delta", delta).
		Msg("🔍 analyzing arbitrage opportunity")

	// 调用 OrderManager.ExecuteArbitrage 执行交易
	execution, err := s.deps.OrderManager.ExecuteArbitrage(
		ctx,
		s.deps.Executor,
		symbol,
		binancePrice,
		bybitPrice,
		1.0, // 默认数量，可从配置读取
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("symbol", symbol).
			Msg("❌ arbitrage execution failed")
		return
	}

	// ✅ 订单执行成功，记录信息
	log.Info().
		Str("symbol", execution.Symbol).
		Str("direction", execution.Direction).
		Float64("quantity", execution.Quantity).
		Str("buy_order_id", execution.BuyOrderID).
		Str("sell_order_id", execution.SellOrderID).
		Float64("expected_profit", execution.ExpectedProfit).
		Float64("expected_profit_rate", execution.ExpectedProfitRate).
		Msg("✓ arbitrage order executed successfully")

	// ✅ 通过 API 验证订单状态
	s.verifyOrderExecution(ctx, symbol, execution)
}

// verifyOrderExecution 通过 API 验证订单执行状态
func (s *Service) verifyOrderExecution(ctx context.Context, symbol string, execution *domainservice.ArbitrageExecution) {
	// 短暂延迟，等待订单在交易所确认
	time.Sleep(500 * time.Millisecond)

	// 验证买单状态（Binance）
	buyStatus, err := s.deps.OrderManager.GetOrderStatus(ctx, "binance", symbol, execution.BuyOrderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("symbol", symbol).
			Str("order_id", execution.BuyOrderID).
			Msg("❌ failed to verify buy order")
		return
	}

	log.Info().
		Str("symbol", symbol).
		Str("order_id", execution.BuyOrderID).
		Str("status", buyStatus.Status).
		Float64("executed_qty", buyStatus.ExecutedQuantity).
		Float64("avg_price", buyStatus.AvgExecutedPrice).
		Msg("✓ buy order verified (Binance)")

	// 验证卖单状态（Bybit）
	sellStatus, err := s.deps.OrderManager.GetOrderStatus(ctx, "bybit", symbol, execution.SellOrderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("symbol", symbol).
			Str("order_id", execution.SellOrderID).
			Msg("❌ failed to verify sell order")
		return
	}

	log.Info().
		Str("symbol", symbol).
		Str("order_id", execution.SellOrderID).
		Str("status", sellStatus.Status).
		Float64("executed_qty", sellStatus.ExecutedQuantity).
		Float64("avg_price", sellStatus.AvgExecutedPrice).
		Msg("✓ sell order verified (Bybit)")

	// ✅ 两个订单都已验证成功
	log.Info().
		Str("symbol", symbol).
		Float64("expected_profit", execution.ExpectedProfit).
		Float64("realized_profit", calculateRealizedProfit(buyStatus, sellStatus)).
		Msg("✅ arbitrage cycle completed and verified")
}

// calculateRealizedProfit 计算实际利润
func calculateRealizedProfit(buyStatus, sellStatus *domainservice.OrderStatus) float64 {
	if buyStatus.ExecutedQuantity == 0 {
		return 0
	}
	// 简化版本：卖出收入 - 买入成本
	return (sellStatus.AvgExecutedPrice - buyStatus.AvgExecutedPrice) * buyStatus.ExecutedQuantity
}
