package monitor

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/application/service"
	domainservice "xarb/internal/domain/service"

	"github.com/rs/zerolog/log"
)

type PriceFeed = port.PriceFeed

type ServiceDeps struct {
	Feeds          []PriceFeed
	Coins          []string // 原始的币种列表，由 feed 实现转换为交易所特定格式
	PrintEveryMin  int
	DeltaThreshold float64
	Exchanges      []string // 要显示的交易所列表（可选），如果为空则显示所有
	Sink           port.Sink
	Repo           port.Repository
	ArbitrageRepo  port.ArbitrageRepository         // 套利仓储
	ArbitrageCalc  *service.ArbitrageCalculator     // 套利计算器
	OrderManager   *domainservice.OrderManager      // Order executor (perpetual)
	Executor       *domainservice.ArbitrageExecutor // 套利分析器
	AccountManager *domainservice.AccountManager    // 账户管理器（可选）
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
	formatter := &Formatter{
		DeltaThreshold: deps.DeltaThreshold,
		Exchanges:      deps.Exchanges, // 使用配置的交易所列表
	}
	return &Service{
		deps:     deps,
		st:       NewState(deps.Coins),
		fmt:      formatter,
		lastBand: make(map[string]int, len(deps.Coins)),
		seenBand: make(map[string]bool, len(deps.Coins)),
		prices:   make(map[string]map[string]float64), // symbol -> exchange -> price
	}
}

func (s *Service) Run(ctx context.Context) error {
	if len(s.deps.Feeds) == 0 {
		return errors.New("no feeds")
	}

	// 记录监控配置
	log.Info().
		Strs("exchanges", s.deps.Exchanges).
		Strs("coins", s.deps.Coins).
		Int("feeds", len(s.deps.Feeds)).
		Float64("delta_threshold", s.deps.DeltaThreshold).
		Msg("✓ Monitor service initialized")

	// 计算将要监控的交易所对
	var pairs []string
	if len(s.deps.Exchanges) >= 2 {
		for i := 0; i < len(s.deps.Exchanges)-1; i++ {
			for j := i + 1; j < len(s.deps.Exchanges); j++ {
				pair := s.deps.Exchanges[i] + " ↔ " + s.deps.Exchanges[j]
				pairs = append(pairs, pair)
			}
		}
		log.Info().
			Strs("pairs", pairs).
			Msg("📊 Cross-exchange arbitrage pair monitoring")
	}

	merged := make(chan port.Tick, 1024)

	// start feeds
	for _, feed := range s.deps.Feeds {
		ch, err := feed.Subscribe(ctx, s.deps.Coins)
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

	// 获取配置的交易所，如果没有配置则使用所有可用的
	exchanges := s.getTradeExchanges(prices)
	if len(exchanges) < 2 {
		log.Warn().
			Str("symbol", symbol).
			Int("exchanges", len(exchanges)).
			Msg("insufficient configured exchanges")
		return
	}

	// 使用前两个交易所执行套利
	ex1, ex2 := exchanges[0], exchanges[1]
	price1, ok1 := prices[ex1]
	price2, ok2 := prices[ex2]

	if !ok1 || !ok2 {
		log.Warn().
			Str("symbol", symbol).
			Str("ex1", ex1).
			Bool("has_ex1", ok1).
			Str("ex2", ex2).
			Bool("has_ex2", ok2).
			Msg("missing required prices")
		return
	}

	// 计算价差和收益率
	priceDiff := price2 - price1
	spreadRate := (priceDiff / price1) * 100

	log.Info().
		Str("symbol", symbol).
		Str("pair", ex1+" ↔ "+ex2).
		Float64("price_"+ex1, price1).
		Float64("price_"+ex2, price2).
		Float64("spread", priceDiff).
		Float64("spread_rate%", spreadRate).
		Msg("🎯 Arbitrage signal detected - ready to execute")

	// 调用 OrderManager 执行套利交易
	execution, err := s.deps.OrderManager.ExecuteArbitrage(
		ctx,
		s.deps.Executor,
		symbol,
		price1,
		price2,
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

// getTradeExchanges 获取要执行交易的交易所列表
// 优先使用 ServiceDeps.Exchanges 配置，否则使用所有可用的交易所前两个
func (s *Service) getTradeExchanges(prices map[string]float64) []string {
	if len(s.deps.Exchanges) > 0 {
		// 使用配置的交易所，但只包含有价格的
		var result []string
		for _, ex := range s.deps.Exchanges {
			if _, ok := prices[ex]; ok {
				result = append(result, ex)
			}
		}
		return result
	}

	// 如果没有配置，使用所有可用的交易所前两个
	result := make([]string, 0, len(prices))
	for ex := range prices {
		result = append(result, ex)
	}
	// 排序确保一致性
	sort.Strings(result)
	if len(result) > 2 {
		result = result[:2]
	}
	return result
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
