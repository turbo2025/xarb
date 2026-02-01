package config

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
			WsURL   string `toml:"ws_url"`
		} `toml:"binance"`

		Bybit struct {
			Enabled bool   `toml:"enabled"`
			WsURL   string `toml:"ws_url"`
		} `toml:"bybit"`
	} `toml:"exchange"`

	Storage struct {
		Enabled bool `toml:"enabled"`

		Redis struct {
			Enabled       bool   `toml:"enabled"`
			Addr          string `toml:"addr"`
			Password      string `toml:"password"`
			DB            int    `toml:"db"`
			Prefix        string `toml:"prefix"`
			TTLSeconds    int    `toml:"ttl_seconds"`
			SignalStream  string `toml:"signal_stream"`
			SignalChannel string `toml:"signal_channel"`
		} `toml:"redis"`

		SQLite struct {
			Enabled bool   `toml:"enabled"`
			Path    string `toml:"path"`
		} `toml:"sqlite"`

		Postgres struct {
			Enabled bool   `toml:"enabled"`
			DSN     string `toml:"dsn"`
		} `toml:"postgres"`
	} `toml:"storage"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.App.PrintEveryMin <= 0 {
		cfg.App.PrintEveryMin = 5
	}
	if cfg.Arbitrage.DeltaThreshold <= 0 {
		cfg.Arbitrage.DeltaThreshold = 5.0
	}

	// storage defaults
	if cfg.Storage.Redis.TTLSeconds <= 0 {
		cfg.Storage.Redis.TTLSeconds = 300
	}
	if strings.TrimSpace(cfg.Storage.Redis.Prefix) == "" {
		cfg.Storage.Redis.Prefix = "xarb"
	}
	if strings.TrimSpace(cfg.Storage.Redis.SignalStream) == "" {
		cfg.Storage.Redis.SignalStream = cfg.Storage.Redis.Prefix + ":signals"
	}
	if strings.TrimSpace(cfg.Storage.Redis.SignalChannel) == "" {
		cfg.Storage.Redis.SignalChannel = cfg.Storage.Redis.Prefix + ":signals:pub"
	}
}

func validate(cfg *Config) error {
	cfg.Symbols.List = normalizeSymbols(cfg.Symbols.List)
	if len(cfg.Symbols.List) == 0 {
		return errors.New("symbols.list is empty")
	}

	if cfg.Exchange.Binance.Enabled && strings.TrimSpace(cfg.Exchange.Binance.WsURL) == "" {
		return errors.New("exchange.binance.ws_url empty but enabled")
	}
	if cfg.Exchange.Bybit.Enabled && strings.TrimSpace(cfg.Exchange.Bybit.WsURL) == "" {
		return errors.New("exchange.bybit.ws_url empty but enabled")
	}

	// storage validation only if storage.enabled
	if cfg.Storage.Enabled {
		if cfg.Storage.Redis.Enabled {
			if strings.TrimSpace(cfg.Storage.Redis.Addr) == "" {
				return errors.New("storage.redis.addr empty but redis enabled")
			}
		}
		if cfg.Storage.SQLite.Enabled {
			if strings.TrimSpace(cfg.Storage.SQLite.Path) == "" {
				return errors.New("storage.sqlite.path empty but sqlite enabled")
			}
		}
		if cfg.Storage.Postgres.Enabled {
			if strings.TrimSpace(cfg.Storage.Postgres.DSN) == "" {
				return errors.New("storage.postgres.dsn empty but postgres enabled")
			}
		}
	}
	return nil
}

func normalizeSymbols(in []string) []string {
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
