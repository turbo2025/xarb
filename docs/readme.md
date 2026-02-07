# xarb - Cryptocurrency Arbitrage Bot

Real-time arbitrage opportunity detector for cryptocurrency trading across multiple exchanges.

## Documentation Index

All documentation lives under the [`docs/`](docs) directory. Start here:

- **[Architecture Deep Dive](ARCHITECTURE.md)** - System design and DDD layers
- **[Arbitrage System Guide](ARBITRAGE.md)** - Trading strategies and mechanisms

## Features

- 🔄 **Multi-Exchange Support**: Binance, Bybit, and extensible to more
- 📊 **Real-time Price Monitoring**: WebSocket connections for instant price updates
- 💡 **Spread Detection**: Identifies profitable arbitrage opportunities
- 💾 **Flexible Storage**: Redis, SQLite, PostgreSQL support
- 📈 **Data Persistence**: Store and analyze historical opportunities
- 🔧 **Configurable**: TOML-based configuration

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

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture.

## Quick Start

### Prerequisites

- Go 1.21+
- Redis (optional, for real-time signals)
- SQLite (included)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/xarb
cd xarb

# Build
make build

# Run
./xarb -config configs/config.toml
```

### Configuration

Edit `configs/config.toml`:

```toml
[app]
print_every_min = 5

[symbols]
list = ["BTCUSDT", "ETHUSDT"]

[exchange.binance]
enabled = true
ws_url = "wss://fstream.binance.com"

[storage.redis]
enabled = true
addr = "127.0.0.1:6379"

[storage.sqlite]
enabled = true
path = "./data/xarb.db"
```

## Development

### Building & Running

```bash
# Build the project
make build

# Run the application
make run

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

- [ ] Prometheus metrics integration
- [ ] Multiple trading strategy support
- [ ] Advanced spread analysis
- [ ] WebSocket connection pooling optimization
- [ ] Distributed execution support
- [ ] REST API for signal queries
- [ ] Web dashboard

## License

MIT License - see LICENSE file for details

## Support

- 📖 [Architecture Guide](docs/ARCHITECTURE.md)
- 💰 [Arbitrage Guide](docs/ARBITRAGE.md)
- 📋 [Project Readme](readme.md)
- 💬 [Issues](https://github.com/yourusername/xarb/issues)

## Acknowledgments

- Built with Go, Redis, SQLite
- Trading data from Binance, Bybit APIs
- Clean Architecture principles from Domain-Driven Design

---

**Status**: Active Development  
**Last Updated**: February 18, 2026
