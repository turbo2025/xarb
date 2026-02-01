package monitor

import (
	"context"
	"errors"
	"time"

	"xarb/internal/application/port"

	"github.com/rs/zerolog/log"
)

type PriceFeed = port.PriceFeed

type ServiceDeps struct {
	Feeds          []PriceFeed
	Symbols        []string
	PrintEveryMin  int
	DeltaThreshold float64
	Sink           port.Sink
	Repo           port.Repository
}

type Service struct {
	deps     ServiceDeps
	st       *State
	fmt      *Formatter
	lastBand map[string]int  // -1/0/+1
	seenBand map[string]bool // 是否已建立基线
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		deps:     deps,
		st:       NewState(deps.Symbols),
		fmt:      NewFormatter(deps.DeltaThreshold),
		lastBand: make(map[string]int, len(deps.Symbols)),
		seenBand: make(map[string]bool, len(deps.Symbols)),
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
			_ = s.deps.Repo.InsertSnapshot(ctx, now.UnixMilli(), line)

		case t := <-merged:
			changed := s.st.Apply(t)
			if changed {
				line := s.fmt.Render(s.st, RenderLive)
				_ = s.deps.Sink.WriteLive(line)
			}

			// persist latest (optional)
			if t.PriceNum > 0 {
				_ = s.deps.Repo.UpsertLatestPrice(ctx, t.Exchange, t.Symbol, t.PriceNum, t.Ts)
			}

			// ---- NEW: threshold crossing detection ----
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
			}

			// 更新 band（即使变回 0 也要更新，才能捕捉下一次穿越）
			s.lastBand[t.Symbol] = band
		}
	}
}
