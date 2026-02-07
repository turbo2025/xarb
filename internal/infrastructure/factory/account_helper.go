package factory

import (
	"context"

	"github.com/rs/zerolog/log"
)

// SpotAccountGetter 现货账户接口
type SpotAccountGetter interface {
	GetBalance(context.Context) (float64, error)
}

// LogSpotBalance 记录现货余额
func LogSpotBalance(ctx context.Context, exchange string, account SpotAccountGetter) {
	if account == nil {
		return
	}
	balance, err := account.GetBalance(ctx)
	if err != nil {
		log.Warn().Err(err).Str("exchange", exchange).Msg("failed to fetch spot balance")
		return
	}
	log.Info().
		Str("exchange", exchange).
		Float64("spot_balance_usdt", balance).
		Msgf("%s spot balance", exchange)
}
