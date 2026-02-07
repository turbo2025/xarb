# xarb - Cryptocurrency Arbitrage Bot

Real-time arbitrage opportunity detector for cryptocurrency trading across multiple exchanges.

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

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture.

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
go build ./cmd/xarb

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

### Using Superpowers

This project uses [Superpowers](https://github.com/obra/superpowers) for AI-driven development.

**Quick start**:
```
Help me plan: [feature description]
```

See [.github/SUPERPOWERS.md](.github/SUPERPOWERS.md) for details.

### Coding Standards

- Go: [.github/skills/GO_CONVENTIONS.md](.github/skills/GO_CONVENTIONS.md)
- Bot: [.github/skills/BOT_CONVENTIONS.md](.github/skills/BOT_CONVENTIONS.md)
- Container: [.github/skills/CONTAINER_PATTERN.md](.github/skills/CONTAINER_PATTERN.md)
- Development: [.github/DEVELOPMENT.md](.github/DEVELOPMENT.md)

### Running Tests

```bash
# Run all tests
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./internal/domain/...

# Check coverage
go test -cover ./... | grep coverage
```

### Code Quality

```bash
# Format code
gofmt -s -w .

# Lint
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
Real-time output of detected opportunities:

```
[SIGNAL] BTC: Binance $45,000 vs Bybit $44,950 (Δ 0.11%)
[SIGNAL] ETH: Binance $2,500 vs Bybit $2,510 (Δ 0.40%)
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

1. Read [DEVELOPMENT.md](.github/DEVELOPMENT.md)
2. Read [GO_CONVENTIONS.md](.github/skills/GO_CONVENTIONS.md)
3. Use Superpowers workflow: `Help me plan: [feature]`
4. Write tests first (TDD)
5. Submit PR with test coverage >= 80%

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

- 📖 [Development Guide](.github/DEVELOPMENT.md)
- 🚀 [Superpowers Setup](.github/SUPERPOWERS.md)
- 📋 [Architecture](.ARCHITECTURE.md)
- 💬 [Issues](https://github.com/yourusername/xarb/issues)

## Acknowledgments

- [Superpowers](https://github.com/obra/superpowers) - AI-driven development framework
- Built with Go, Redis, SQLite
- Trading data from Binance, Bybit APIs

---

**Status**: Active Development

Last Updated: February 2026
