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
	Executor       *domainservice.ArbitrageExecutor // 套利分析器
	// AccountManager *domainservice.AccountManager    // 账户管理器（可选）
}

type Service struct {
	deps     ServiceDeps     // 依赖项（Feeds、Sink、Repository等）
	st       *State          // 监控状态（当前价格、对价差等）
	fmt      *Formatter      // 输出格式化器
	lastBand map[string]int  // 分布带记忆：-1（低）、0（中）、+1（高）
	seenBand map[string]bool // 是否已建立基线（防止冷启动误报）

	// 用于订单执行的价格缓存
	pricesLock sync.RWMutex                  // 并发读写锁
	prices     map[string]map[string]float64 // 价格缓存：symbol → exchange → price
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
		go func(f port.PriceFeed, in <-chan port.Tick) {
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-in:
					if !ok {
						return
					}
					// 使用 feed 的转换器将交易对转换为币种
					coin := f.Symbol2Coin(t.Symbol)
					if coin == "" {
						log.Warn().Str("feed", f.Name()).Str("symbol", t.Symbol).Msg("failed to convert symbol to coin")
						continue
					}
					// 创建币种级别的 Tick
					tCoin := t
					tCoin.Symbol = coin
					log.Debug().Str("feed", f.Name()).Str("symbol", t.Symbol).Str("coin", coin).Str("price", t.PriceStr).Msg("converted symbol to coin")
					merged <- tCoin
				}
			}
		}(feed, ch)

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
			// 使用 feed 的转换器将交易对转换为币种
			coin := t.Symbol // 现在 Symbol 已经是币种了（在前面 goroutine 中转换过）
			price := t.PriceStr

			// 保存价格到缓存
			s.pricesLock.Lock()
			if s.prices[coin] == nil {
				s.prices[coin] = make(map[string]float64)
			}
			s.prices[coin][t.Exchange] = t.PriceNum
			s.pricesLock.Unlock()

			log.Debug().Str("coin", coin).Str("exchange", t.Exchange).Str("price", price).Msg("received tick")

			changed := s.st.Apply(t)
			if changed {
				log.Debug().Str("coin", coin).Msg("price changed")
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
				// payload := s.fmt.Render(s.st, RenderSnapshot) // 用快照格式（无 \r / 清行）
				// ⚠️ 信号直接打到 console（一次）
				_ = s.deps.Sink.NewLine()
				log.Warn().
					Str("symbol", t.Symbol).
					Float64("delta", delta).
					Int("band", band).
					Float64("threshold", s.deps.DeltaThreshold).
					Msg("arbitrage signal detected")

				// 发送飞书通知（如果配置了飞书）
				// s.sendFeishuSignal(t.Symbol, delta, payload)

				// ✅ 新增：检测到套利机会，执行订单！
				if s.deps.Executor != nil {
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

// calculateRealizedProfit 计算实际利润
func calculateRealizedProfit(buyStatus, sellStatus *domainservice.OrderStatus) float64 {
	if buyStatus.ExecutedQuantity == 0 {
		return 0
	}
	// 简化版本：卖出收入 - 买入成本
	return (sellStatus.AvgExecutedPrice - buyStatus.AvgExecutedPrice) * buyStatus.ExecutedQuantity
}

// sendFeishuSignal 发送套利信号到飞书
func (s *Service) sendFeishuSignal(symbol string, delta float64, payload string) {
	// 使用类型断言检查是否是飞书 Sink
	type FeishuSender interface {
		SendSignal(symbol string, delta float64, payload string) error
	}

	if feishuSink, ok := s.deps.Sink.(FeishuSender); ok {
		if err := feishuSink.SendSignal(symbol, delta, payload); err != nil {
			log.Error().
				Err(err).
				Str("symbol", symbol).
				Float64("delta", delta).
				Msg("failed to send feishu signal")
		}
	}
}
