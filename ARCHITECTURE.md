# Binance WebSocket Arbitrage Monitor - DDD Architecture

A clean, extensible crypto arbitrage monitoring system following Domain-Driven Design (DDD) principles.

## Project Structure

```
binance-ws/
├── domain/                    # Core business logic (pure logic, no dependencies)
│   ├── board.go              # Board aggregate root - price tracking
│   └── price.go              # Price value objects
│
├── application/              # Application services (orchestration layer)
│   └── service.go            # Use case implementations
│
├── infrastructure/           # Technical implementations
│   ├── exchange/            # Exchange adapters
│   │   ├── exchange.go      # Common exchange interfaces & utilities
│   │   ├── binance.go       # Binance adapter
│   │   └── bybit.go         # Bybit adapter
│   └── storage/             # Data persistence layer
│       └── storage.go        # Repository interfaces & implementations
│
├── presentation/            # UI/Display layer
│   └── renderer.go          # Terminal rendering
│
├── config.go                # Configuration loading
├── config.toml              # Configuration file
├── main.go                  # Application entry point
├── go.mod                   # Module definition
└── go.sum                   # Dependency checksums
```

## Architecture Layers

### 1. **Domain Layer** (`domain/`)
- Contains pure business logic independent of any framework
- Implements aggregates: `Board` (price tracking across exchanges)
- Value objects: `PriceState`, `SymbolState`, `Direction`
- No external dependencies (except stdlib)

### 2. **Application Layer** (`application/`)
- Orchestrates domain objects and infrastructure services
- Contains use cases:
  - `PriceUpdateService`: Handles price updates from exchanges
  - `SnapshotPrinterService`: Manages periodic reporting
  - `ExchangeRunnerService`: Manages exchange connections
- Adapts infrastructure to domain needs via interfaces

### 3. **Infrastructure Layer** (`infrastructure/`)

#### Exchange Adapters (`infrastructure/exchange/`)
- Implements WebSocket connections to exchanges
- Common interface `Exchange` for all adapters
- Automatic reconnection with exponential backoff
- Built-in WebSocket ping/pong and heartbeat

#### Storage Layer (`infrastructure/storage/`)
- Repository pattern interfaces for data persistence
- Currently provides in-memory implementations
- Ready for extension to:
  - PostgreSQL
  - SQLite3
  - Redis
  - MongoDB

### 4. **Presentation Layer** (`presentation/`)
- Terminal rendering with ANSI colors
- Decoupled from business logic
- Easy to extend for other output formats (JSON, HTTP, etc.)

## Future Extension Points

### Adding a New Exchange
1. Create `infrastructure/exchange/newexchange.go`
2. Implement `Exchange` interface
3. Add configuration to `config.toml`
4. Update `main.go` to instantiate the adapter

### Adding Persistence
1. Implement `storage.PriceRepository` for your database
   - PostgreSQL: `infrastructure/storage/postgres.go`
   - SQLite3: `infrastructure/storage/sqlite.go`
   - Redis: `infrastructure/storage/redis.go`
2. Update `NewInMemoryStorage()` call in `main.go`

### Adding New Output Formats
1. Create new renderer in `presentation/`
2. Implement renderer interface
3. Integrate into application layer

## Key Design Patterns

### Dependency Inversion
- Domain layer defines interfaces
- Infrastructure implements them
- Application layer coordinates via interfaces

### Repository Pattern
- Abstracts data storage
- Easy to mock for testing
- Supports multiple implementations

### Adapter Pattern
- Each exchange is an adapter
- Uniform interface for different APIs
- Easy to add new exchanges

### Logger Adapter
- Decouples application from logging framework
- Simple interface for different logging backends

## Building & Running

```bash
# Build
go build -v

# Run
./binance-ws -config config.toml
```

## Configuration

Edit `config.toml`:

```toml
[app]
print_every_min = 5          # Snapshot frequency

[symbols]
list = ["BTCUSDT", "ETHUSDT"]

[arbitrage]
delta_threshold = 5.0        # Price difference threshold for highlighting

[exchange.binance]
enabled = true
ws_url = "wss://fstream.binance.com"

[exchange.bybit]
enabled = true
ws_url = "wss://stream.bybit.com/v5/public/linear"
```

## Testing & Modularity

Each layer can be tested independently:
- **Domain**: Pure unit tests
- **Application**: Mock infrastructure services
- **Infrastructure**: Integration tests
- **Presentation**: Output format verification

## Benefits of This Architecture

✅ **Separation of Concerns**: Each layer has single responsibility
✅ **Testability**: Easy to mock and test each component
✅ **Extensibility**: Simple to add exchanges, storage, or outputs
✅ **Maintainability**: Clear structure and organization
✅ **Scalability**: Ready for databases and complex features
✅ **Flexibility**: Swap implementations without changing business logic
