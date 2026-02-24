package config

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// TradeConfig 交易类型配置（Spot或Perpetual）
type TradeConfig struct {
	Enabled bool   `toml:"enabled"`
	HTTP    string `toml:"http"`
	WS      string `toml:"ws"`
}

// ExchangeConfig 交易所配置（通用格式）
type ExchangeConfig struct {
	Enabled    bool        `toml:"enabled"`
	APIKey     string      `toml:"api_key"`
	SecretKey  string      `toml:"secret_key"`
	Passphrase string      `toml:"passphrase"` // 仅 OKX、Bitget 需要
	Spot       TradeConfig `toml:"spot"`
	Perpetual  TradeConfig `toml:"perpetual"`
	// 以下字段保留用于向后兼容
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
		Coins []string `toml:"coins"` // 币种列表 (e.g., ["BTC", "ETH", "SOL"])
		Quote string   `toml:"quote"` // 计价货币 (e.g., "USDT")
	} `toml:"symbols"`

	Arbitrage struct {
		DeltaThreshold float64 `toml:"delta_threshold"`
	} `toml:"arbitrage"`

	Monitor struct {
		Exchanges []string `toml:"exchanges"` // 要监控的交易所列表（可选，如果为空则监控所有enabled的）
	} `toml:"monitor"`

	Exchanges map[string]ExchangeConfig `toml:"exchanges"`

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

	Message struct {
		Feishu []FeishuConfig `toml:"feishu"`
	} `toml:"message"`
}

type FeishuConfig struct {
	Channel string `toml:"channel"` // signal, order, pnl, market 等
	Webhook string `toml:"webhook"` // 飞书机器人 webhook 地址
	Secret  string `toml:"secret"`  // 飞书签名密钥
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
	if cfg.Redis.TTLSeconds <= 0 {
		cfg.Redis.TTLSeconds = 300
	}
	if strings.TrimSpace(cfg.Redis.Prefix) == "" {
		cfg.Redis.Prefix = "xarb"
	}
	if strings.TrimSpace(cfg.Redis.SignalStream) == "" {
		cfg.Redis.SignalStream = cfg.Redis.Prefix + ":signals"
	}
	if strings.TrimSpace(cfg.Redis.SignalChannel) == "" {
		cfg.Redis.SignalChannel = cfg.Redis.Prefix + ":signals:pub"
	}
}

func validate(cfg *Config) error {
	// 处理新的 coins + quote 格式
	if len(cfg.Symbols.Coins) == 0 {
		return errors.New("symbols.coins is empty")
	}
	if strings.TrimSpace(cfg.Symbols.Quote) == "" {
		return errors.New("symbols.quote is empty")
	}
	// 验证所有启用的交易所都有必要的配置
	for exchangeName, exchCfg := range cfg.Exchanges {
		if !exchCfg.Enabled {
			continue
		}
		// 至少需要启用spot或perpetual中的一个
		if !exchCfg.Spot.Enabled && !exchCfg.Perpetual.Enabled {
			return fmt.Errorf("exchange.%s: at least one of spot or perpetual must be enabled", exchangeName)
		}
		// 检查启用的交易类型是否有有效的WebSocket URL
		if exchCfg.Spot.Enabled && strings.TrimSpace(exchCfg.Spot.WS) == "" {
			return fmt.Errorf("exchange.%s.spot.ws empty but spot enabled", exchangeName)
		}
		if exchCfg.Perpetual.Enabled && strings.TrimSpace(exchCfg.Perpetual.WS) == "" {
			return fmt.Errorf("exchange.%s.perpetual.ws empty but perpetual enabled", exchangeName)
		}
	}

	if cfg.Redis.Enabled {
		if strings.TrimSpace(cfg.Redis.Addr) == "" {
			return errors.New("redis.addr empty but redis enabled")
		}
	}
	if cfg.SQLite.Enabled {
		if strings.TrimSpace(cfg.SQLite.Path) == "" {
			return errors.New("sqlite.path empty but sqlite enabled")
		}
	}
	if cfg.Postgres.Enabled {
		if strings.TrimSpace(cfg.Postgres.DSN) == "" {
			return errors.New("postgres.dsn empty but postgres enabled")
		}
	}
	return nil
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

// GetEnabledExchanges 获取所有enabled的交易所名称列表
// 如果Monitor.Exchanges已配置，则使用该列表中enabled的交易所
// 否则，返回所有enabled的交易所，按字母顺序排列
func (c *Config) GetEnabledExchanges() []string {
	var enabledExchanges []string

	// 如果Monitor中配置了特定的交易所，检查这些交易所是否启用
	if len(c.Monitor.Exchanges) > 0 {
		for _, exName := range c.Monitor.Exchanges {
			if cfg, ok := c.Exchanges[exName]; ok && cfg.Enabled {
				enabledExchanges = append(enabledExchanges, exName)
			}
		}
		return enabledExchanges
	}

	// 否则，返回所有enabled的交易所
	// 为保证顺序一致，我们需要按字母顺序返回
	var allExchanges []string
	for name, cfg := range c.Exchanges {
		if cfg.Enabled {
			allExchanges = append(allExchanges, name)
		}
	}
	// 排序确保顺序一致
	sort.Strings(allExchanges)
	return allExchanges
}
