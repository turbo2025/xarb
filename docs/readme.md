# xarb - Cryptocurrency Arbitrage Bot

Real-time arbitrage opportunity detector for cryptocurrency trading across multiple exchanges.

## Documentation Index

All documentation lives under the [`docs/`](docs) directory. Start here:

- **[Architecture Deep Dive](ARCHITECTURE.md)** - System design and DDD layers
- **[Arbitrage System Guide](ARBITRAGE.md)** - Trading strategies and mechanisms  
- **[WebSocket Architecture](WEBSOCKET_ARCHITECTURE.md)** - Real-time data streaming design

## Features

- 🔄 **Multi-Exchange Support**: Binance, Bybit, OKX, Bitget with unified API
- 📊 **Real-time Price Monitoring**: WebSocket connections for instant price updates
- 💡 **Spread Detection**: Automatic arbitrage opportunity identification
- 🚀 **Smart Order Execution**: Multi-exchange order execution with deduplication
- 💰 **Feishu Notifications**: Real-time signal alerts via Feishu bot
- 💾 **Flexible Storage**: Redis, SQLite, PostgreSQL support
- 📈 **Data Persistence**: Store and analyze historical opportunities
- 🔧 **Configurable**: TOML-based configuration with coins + quote format

## Architecture

```
┌─────────────────────────────────────┐
│       User Interface                │
│   (Console, HTTP, Dashboard)        │
├─────────────────────────────────────┤
│    Application Layer                │
│   (Monitoring, Strategies)          │
├─────────────────────────────────────┤
│     Domain Layer                    │
│  (Core Business Logic)              │
├─────────────────────────────────────┤
│   Infrastructure Layer              │
│ (Exchanges, Storage, Config)        │
└─────────────────────────────────────┘
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture.

## Key Components

### Configuration

XARB uses a flexible TOML-based configuration with symbol management:

```toml
[symbols]
coins = ["BTC", "ETH", "SOL"]
quote = "USDT"

[arbitrage]
delta_threshold = 5.0  # 5% spread threshold

[monitor]
exchanges = ["binance", "bybit", "okx"]  # Cross-exchange pairs

[message.feishu]
[[message.feishu]]
channel = "signal"
webhook = "https://open.feishu.cn/..."
secret = "xxx"
```

### Service Architecture

```
ServiceContext (Central Hub)
├── Exchanges (HTTP API Clients)
│   ├── Binance (Spot + Perpetual)
│   ├── Bybit (Spot + Perpetual)
│   ├── OKX (Spot + Perpetual)
│   └── Bitget (Spot + Perpetual)
├── WebSocket Manager
│   └── Real-time Price Feeds (all exchanges)
├── Monitor Service
│   ├── Price State Management
│   ├── Spread Calculation
│   └── Feishu Notifications
└── Storage Layer
    ├── SQLite (default)
    ├── Redis (optional)
    └── PostgreSQL (optional)
```

## Quick Start

### Prerequisites

- Go 1.21+
- Redis (optional, for signal streaming)
- SQLite (included)

### Installation

```bash
# Clone repository
git clone https://github.com/yourusername/xarb.git
cd xarb

# Build
go build ./cmd/xarb

# Run
./xarb
```

### Configuration

1. Edit `configs/config.toml`:

```toml
[app]
print_every_min = 5

[symbols]
coins = ["BTC", "ETH", "SOL"]
quote = "USDT"

[arbitrage]
delta_threshold = 5.0

[exchanges.binance]
enabled = true
perpetual_ws_url = "wss://fstream.binance.com/ws"
api_key = "your_api_key"
secret_key = "your_secret_key"

[exchanges.bybit]
enabled = true
perpetual_ws_url = "wss://stream.bybit.com/v5/public/linear"
api_key = "your_api_key"
secret_key = "your_secret_key"

[exchanges.okx]
enabled = true
perpetual_ws_url = "wss://ws.okx.com:8443/ws/v5/public"
api_key = "your_api_key"
secret_key = "your_secret_key"
passphrase = "your_passphrase"

[sqlite]
enabled = true
path = "./data/xarb.db"

# Optional: Feishu notifications
[message.feishu]
[[message.feishu]]
channel = "signal"
webhook = "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
secret = "your_secret_key"
```

2. Run:

```bash
./xarb
```

### Running Tests

```bash
go test ./...
```

## Core Concepts

### Symbol Management

XARB uses a unified symbol format across all exchanges:
- **Config**: Raw coins list (`["BTC", "ETH", "SOL"]`) + quote currency (`"USDT"`)
- **Conversion**: Each exchange has a `SymbolConverter` that handles format mapping
  - Binance/Bybit/Bitget: `BTC` → `BTCUSDT`
  - OKX: `BTC` → `BTC-USDT-SWAP`

### Arbitrage Detection Flow

```
WebSocket Tick (exchange format)
  ↓
Symbol2Coin() conversion
  ↓
State.Apply() - price updates
  ↓
DeltaBand calculation - spread detection
  ↓
Band threshold crossing
  ↓
Signal generation → Feishu notification + Order execution
```

## Development

### Building

```bash
# Build
go build ./cmd/xarb

# Run tests
make test
```

### Code Quality

```bash
# Format code
gofmt -s -w ./internal ./cmd

# Lint (if golangci-lint is installed)
golangci-lint run

# Verify dependencies
go mod verify
```

## Project Structure

```
.
├── cmd/                          # Application entry points
│   └── xarb/                     # Main bot application
├── configs/                      # Configuration files
├── data/                         # Data storage (sqlite, etc)
├── internal/
│   ├── application/              # Use cases, business logic
│   │   └── usecase/monitor/      # Main monitoring service
│   ├── domain/                   # Core business logic
│   │   ├── model/                # Domain models
│   │   └── service/              # Business services
│   ├── infrastructure/           # External dependencies
│   │   ├── config/               # Configuration loading
│   │   ├── container/            # Dependency injection
│   │   ├── exchange/             # Exchange integrations
│   │   ├── logger/               # Logging
│   │   └── storage/              # Data persistence
│   └── interfaces/               # External interfaces
│       ├── console/              # CLI output
│       └── http/                 # REST API
├── tests/                        # Integration tests
├── .github/
│   ├── skills/                   # Development guidelines
│   ├── designs/                  # Design documents
│   ├── DEVELOPMENT.md            # Development workflow
│   └── SUPERPOWERS.md            # Superpowers guide
├── go.mod, go.sum                # Go dependencies
└── readme.md                     # This file
```

## Key Components

### Monitor Service
Real-time arbitrage opportunity monitoring across multiple exchanges.

- **Location**: `internal/application/usecase/monitor/`
- **Entry**: `service.go`
- **Tests**: `*_test.go` files

### Price Feeds
WebSocket connections to exchange APIs.

- **Binance**: `internal/infrastructure/exchange/binance/`
- **Bybit**: `internal/infrastructure/exchange/bybit/`
- **Interface**: `internal/application/port/pricefeed.go`

### Storage Layer
Persistent data storage with multiple backends.

- **Redis**: `internal/infrastructure/storage/redis/`
- **SQLite**: `internal/infrastructure/storage/sqlite/`
- **Postgres**: `internal/infrastructure/storage/postgres/`

### Dependency Container
Centralized dependency management.

- **Location**: `internal/infrastructure/container/`
- **Usage**: See [CONTAINER_PATTERN.md](.github/skills/CONTAINER_PATTERN.md)

## Configuration

### Environment Variables

Sensitive configuration can be overridden by environment variables:

```bash
export REDIS_PASSWORD=your_password
export DB_DSN=postgres://user:pass@localhost/xarb
```

### Config Validation

All configuration is validated at startup:

```bash
./xarb -config configs/config.toml
# Logs configuration load status and validates all required fields
```

## API & Interfaces

### Console Output
Real-time monitoring of detected opportunities:

```
[MONITOR] Starting arbitrage monitoring...
[INFO] Binance futures balance: 1000.00 USDT
[INFO] Bybit futures balance: 1000.00 USDT
[PRICE] BTC: Binance $45,000 | Bybit $44,950 (Δ 0.11%)
[PRICE] ETH: Binance $2,500 | Bybit $2,510 (Δ -0.40%)
[SIGNAL] Arbitrage opportunity detected - BTC spread increased
```

### HTTP Server (Optional)
Health checks and metrics endpoints (if enabled):

```
GET /health         - Health check
GET /metrics        - Prometheus metrics (if enabled)
```

## Monitoring & Alerts

### Metrics
- Price spreads per trading pair
- Update latency per exchange
- Arbitrage opportunity frequency
- Data accuracy and coverage

### Health Checks
- Exchange connectivity
- Storage backend availability
- Message processing capacity

See [BOT_CONVENTIONS.md](.github/skills/BOT_CONVENTIONS.md#监控告警) for details.

## Contributing

1. Read [Architecture Guide](docs/ARCHITECTURE.md)
2. Read [Arbitrage Guide](docs/ARBITRAGE.md)
3. Write tests first (TDD approach)
4. Follow Go conventions (gofmt, clear naming)
5. Ensure tests pass: `make test`

## Roadmap

**Current (v0.1)**:
- ✅ Multi-exchange WebSocket price monitoring
- ✅ Real-time spread detection
- ✅ Feishu notifications
- ✅ SQLite/Redis/PostgreSQL support
- ✅ Symbol format conversion

**Planned (v0.2)**:
- [ ] Order execution with risk management
- [ ] Funding rate arbitrage strategy
- [ ] Advanced spread analysis
- [ ] REST API for queries
- [ ] Web dashboard
- [ ] Distributed deployment support

## Status

**Development Status**: 🚀 Active Development  
**Current Version**: 0.1.0-beta  
**Last Updated**: February 23, 2026

### Working Features
- ✅ Real-time price monitoring from 4 exchanges
- ✅ Automatic spread detection
- ✅ Feishu bot notifications
- ✅ Configurable monitoring parameters
- ✅ Multiple storage backends

### Known Limitations
- Order execution framework in progress
- Risk management system under development
- Funding rate arbitrage not yet implemented

## Support & Documentation

- 📖 [Architecture Guide](ARCHITECTURE.md) - Complete system design
- 💰 [Arbitrage Guide](ARBITRAGE.md) - Trading strategies
- 🔌 [WebSocket Architecture](WEBSOCKET_ARCHITECTURE.md) - Real-time data design
- 💬 [Issues](https://github.com/yourusername/xarb/issues) - Report bugs

## Contributing

Contributions welcome! Areas needing help:
- Order execution implementation
- Risk management logic
- New exchange integrations
- Documentation improvements
- Test coverage

## Acknowledgments

- Built with [Go](https://golang.org/) 1.21+
- Storage: [SQLite](https://www.sqlite.org/), [Redis](https://redis.io/), [PostgreSQL](https://www.postgresql.org/)
- Exchanges: [Binance](https://www.binance.com/), [Bybit](https://www.bybit.com/), [OKX](https://www.okx.com/), [Bitget](https://www.bitget.com/)
- Architecture: [Domain-Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design)
- Notifications: [Feishu Bot API](https://open.feishu.cn/)

## License

MIT License

---

**Start here**: [Architecture Deep Dive](ARCHITECTURE.md) | [Quick Start](#quick-start)

