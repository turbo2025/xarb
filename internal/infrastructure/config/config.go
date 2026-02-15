package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// ExchangeConfig 交易所配置（通用格式）
type ExchangeConfig struct {
	Enabled          bool   `toml:"enabled"`
	APIKey           string `toml:"api_key"`
	SecretKey        string `toml:"secret_key"`
	Passphrase       string `toml:"passphrase"` // 仅 OKX、Bitget 需要
	SpotHttpURL      string `toml:"spot_http_url"`
	SpotWsURL        string `toml:"spot_ws_url"`
	PerpetualHttpURL string `toml:"perpetual_http_url"`
	PerpetualWsURL   string `toml:"perpetual_ws_url"`
}

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

	Exchanges map[string]ExchangeConfig `toml:"exchanges"`

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

	// 验证所有启用的交易所都有必要的配置
	for exchangeName, exchCfg := range cfg.Exchanges {
		if !exchCfg.Enabled {
			continue
		}
		if strings.TrimSpace(exchCfg.PerpetualWsURL) == "" {
			return fmt.Errorf("exchange.%s.perpetual_ws_url empty but enabled", exchangeName)
		}
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

// Exported types for programmatic configuration
type StorageConfig struct {
	Enabled  bool
	SQLite   SQLiteConfig
	Redis    RedisConfig
	Postgres PostgresConfig
}

type SQLiteConfig struct {
	Enabled bool
	Path    string
}

type RedisConfig struct {
	Enabled       bool
	Addr          string
	Password      string
	DB            int
	Prefix        string
	TTLSeconds    int
	SignalStream  string
	SignalChannel string
}

type PostgresConfig struct {
	Enabled bool
	DSN     string
}
