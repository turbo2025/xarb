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
	deps ServiceDeps
	st   *State
	fmt  *Formatter
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		deps: deps,
		st:   NewState(deps.Symbols),
		fmt:  NewFormatter(deps.DeltaThreshold),
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
			// optional: persist latest
			if t.PriceNum > 0 {
				_ = s.deps.Repo.UpsertLatestPrice(ctx, t.Exchange, t.Symbol, t.PriceNum, t.Ts)
			}
		}
	}
}
