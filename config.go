package main

import (
	"errors"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	App struct {
		PrintEveryMin int `toml:"print_every_min"`
	} `toml:"app"`

	Symbols struct {
		List []string `toml:"list"`
	} `toml:"symbols"`

	Arbitrage struct {
		DeltaThreshold float64 `toml:"delta_threshold"`
	} `toml:"arbitrage"`

	Exchange struct {
		Binance struct {
			Enabled bool   `toml:"enabled"`
			WsURL   string `toml:"ws_url"` // e.g. wss://fstream.binance.com
		} `toml:"binance"`

		Bybit struct {
			Enabled bool   `toml:"enabled"`
			WsURL   string `toml:"ws_url"` // e.g. wss://stream.bybit.com/v5/public/linear
		} `toml:"bybit"`
	} `toml:"exchange"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	ApplyDefaults(&cfg)
	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ApplyDefaults(cfg *Config) {
	if cfg.App.PrintEveryMin <= 0 {
		cfg.App.PrintEveryMin = 5
	}
	if cfg.Arbitrage.DeltaThreshold <= 0 {
		cfg.Arbitrage.DeltaThreshold = 5.0
	}
	// if ws_url missing but enabled, validation will catch it
}

func ValidateConfig(cfg *Config) error {
	cfg.Symbols.List = NormalizeSymbols(cfg.Symbols.List)
	if len(cfg.Symbols.List) == 0 {
		return errors.New("symbols.list is empty")
	}

	// Exchanges: allow disabling both, but then nothing will update.
	if cfg.Exchange.Binance.Enabled && strings.TrimSpace(cfg.Exchange.Binance.WsURL) == "" {
		return errors.New("exchange.binance.ws_url is empty but binance enabled")
	}
	if cfg.Exchange.Bybit.Enabled && strings.TrimSpace(cfg.Exchange.Bybit.WsURL) == "" {
		return errors.New("exchange.bybit.ws_url is empty but bybit enabled")
	}
	return nil
}

func NormalizeSymbols(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}
