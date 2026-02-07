# XARB Architecture - Domain-Driven Design

A clean, extensible crypto arbitrage system following Domain-Driven Design (DDD) principles.

## Architecture Layers

```
┌────────────────────────────────────┐
│      Interfaces Layer              │
│   (Console, HTTP APIs)             │
├────────────────────────────────────┤
│    Application Layer               │
│  (Use Cases, Orchestration)        │
├────────────────────────────────────┤
│      Domain Layer                  │
│   (Business Rules & Logic)         │
├────────────────────────────────────┤
│   Infrastructure Layer             │
│ (Exchanges, Storage, Config)       │
└────────────────────────────────────┘
```

### 1. Domain Layer (`internal/domain/`)

Pure business logic independent of frameworks and infrastructure.

**Key Components:**

- **Models** (`domain/model/`)
  - `FuturesPrice` - Price data with funding rates
  - `Symbol` - Trading pair definitions
  - `SpreadArbitrage`, `FundingArbitrage` - Opportunity models

- **Services** (`domain/service/`)
  - `ArbitrageExecutor` - Core profit calculation engine
  - `OrderManager` - Order execution logic
  - `RiskManager` - Risk validation and management
  - `MarginManager` - Margin requirement calculations
  - `DuplicateOrderGuard` - Prevent duplicate orders
  - `TradeTypeManager` - Trade type handling

**Characteristics:**
- No dependencies on frameworks or external libraries (except stdlib)
- Can be tested in isolation with pure unit tests
- Contains complex business algorithms and decision logic
- Defines interfaces that infrastructure must implement

### 2. Application Layer (`internal/application/`)

Business processes and use case implementations that orchestrate domain services.

**Key Components:**

- **Ports** (`application/port/`)
  - Interface definitions for dependency injection
  - `ArbitrageRepository` - Data persistence interface
  - `PriceFeed` - Price update interface
  - `EventBus` - Event distribution

- **Services** (`application/service/`)
  - `ArbitrageService` - Opportunity scanning
  - `PriceService` - Price data management
  - `PositionService` - Position tracking
  - `FundingRateSyncer` - Funding rate updates
  - `SnapshotService` - Data snapshots
  - `SignalService` - Signal generation
  - `ArbitrageCalculator` - Calculation utilities

- **Use Cases** (`application/usecase/`)
  - `monitor/` - Main monitoring service
    - `service.go` - Core monitoring logic
    - `formatter.go` - Output formatting
    - `state.go` - State management
    - `types.go` - Type definitions

- **Container** (`application/container/`)
  - `container.go` - Dependency injection container
  - Centralizes service instantiation and wiring

**Characteristics:**
- Depends on `port` interfaces (dependency inversion)
- Relatively thin layer, mostly coordinating domain and infrastructure
- No business rules here, only process flow
- Easy to extend with new use cases

### 3. Infrastructure Layer (`internal/infrastructure/`)

Technical implementations and external dependencies.

**Key Components:**

- **Exchange Adapters** (`infrastructure/exchange/`)
  - `binance/` - Binance REST & WebSocket clients
    - `futures_account_client.go` - Account info
    - `futures_order_client.go` - Order management
    - `futures_position_client.go` - Position queries
    - `ws_client.go` - WebSocket connections
  - `bybit/` - Bybit REST & WebSocket clients
    - `linear_account_client.go` - Account info
    - `linear_order_client.go` - Order management
    - `linear_position_client.go` - Position queries
    - `ws_client.go` - WebSocket connections
  - Similar adapters for OKX, Bitget, etc.

- **Storage Layer** (`infrastructure/storage/`)
  - `sqlite/` - SQLite implementation
  - `postgres/` - PostgreSQL implementation
  - `redis/` - Redis implementation
  - Common interface: `Repository`

- **Configuration** (`infrastructure/config/`)
  - TOML configuration loading
  - Environment variable overrides
  - Validation

- **Logger** (`infrastructure/logger/`)
  - Structured logging with zerolog
  - Decoupled from business logic

### 4. Interfaces Layer (`internal/interfaces/`)

External-facing interfaces and adapters.

**Components:**

- **Console** (`interfaces/console/`)
  - CLI output formatting
  - Terminal rendering

- **HTTP** (`interfaces/http/`)
  - REST API endpoints (optional)
  - Health checks and metrics

## Data Flow

### Price Update Flow
```
Exchange WebSocket
    ↓
infrastructure/exchange/{exchange}/ws_client.go
    ↓
application/port/pricefeed.go (interface)
    ↓
application/service/price_service.go
    ↓
application/usecase/monitor/service.go
```

### Arbitrage Detection Flow
```
Price Update
    ↓
application/usecase/monitor/service.go
    ↓
application/service/arbitrage_service.go
    ↓
domain/service/arbitrage_executor.go (business logic)
    ↓
application/port/arbitrage.go (interface)
    ↓
infrastructure/storage/{backend}/ (persist)
```

### Order Execution Flow
```
Arbitrage Signal
    ↓
domain/service/risk_manager.go (validate)
    ↓
domain/service/order_manager.go (execute)
    ↓
infrastructure/exchange/{exchange}/futures_order_client.go
    ↓
Exchange REST API
```

## Key Design Patterns

### 1. Dependency Inversion Principle (DIP)
- Domain layer defines interfaces (ports)
- Application uses these interfaces
- Infrastructure implements them
- Allows easy testing and swapping implementations

### 2. Repository Pattern
- Abstracts data persistence
- Single interface, multiple implementations
- Supports SQLite, PostgreSQL, Redis seamlessly

### 3. Adapter Pattern
- Each exchange is an adapter
- Uniform interface (`OrderClient`, `AccountClient`, etc.)
- Easy to add new exchanges without changing business logic

### 4. Service Locator / Dependency Container
- `application/container/container.go`
- Centralizes service creation
- Makes testing easier with mock containers

### 5. Strategy Pattern
- Different trading strategies (spread, funding, etc.)
- Easy to add new strategies without changing core logic

## Project Structure

```
xarb/
├── cmd/
│   └── xarb/
│       └── main.go                 # Entry point
├── configs/
│   └── config.toml                 # Configuration
├── data/                           # Database files
├── docs/                           # Documentation
├── internal/
│   ├── application/
│   │   ├── container/              # DI container
│   │   ├── port/                   # Interface definitions
│   │   ├── service/                # Business logic coordination
│   │   └── usecase/                # Use case implementations
│   │       └── monitor/            # Monitoring use case
│   ├── domain/
│   │   ├── model/                  # Domain models
│   │   └── service/                # Core business logic
│   ├── infrastructure/
│   │   ├── config/                 # Configuration loading
│   │   ├── container/              # Container implementation
│   │   ├── exchange/               # Exchange adapters
│   │   ├── factory/                # Factory helpers
│   │   ├── logger/                 # Logging
│   │   ├── storage/                # Data persistence
│   │   └── svc/                    # Service utilities
│   └── interfaces/
│       ├── console/                # CLI output
│       └── http/                   # REST API
├── go.mod                          # Go modules
├── go.sum                          # Dependencies
└── Makefile                        # Build script
```

## Extension Points

### Adding a New Exchange

1. Create new adapter in `infrastructure/exchange/{exchange}/`
2. Implement required client interfaces:
   - `AccountClient` - Account information
   - `OrderClient` - Order management
   - `PositionClient` - Position queries
3. Update factory in `infrastructure/factory/`
4. Add configuration to `config.toml`

### Adding New Storage Backend

1. Implement `repository.Repository` interface
2. Create in `infrastructure/storage/{backend}/`
3. Update container in `application/container/`
4. Add configuration option in `config.toml`

### Adding New Use Case

1. Create service in `application/service/`
2. Implement in `application/usecase/`
3. Wire into container via `application/container/`
4. Expose via `interfaces/` layer if needed

## Testing Strategy

- **Domain Services**: Pure unit tests, no mocks needed
- **Application Services**: Mock infrastructure via ports interface
- **Infrastructure**: Integration tests with test databases
- **Use Cases**: End-to-end tests with mock services

## Configuration

Edit `configs/config.toml`:

```toml
[app]
print_every_min = 1

[symbols]
list = ["BTCUSDT", "ETHUSDT"]

[exchange.binance]
enabled = true
ws_url = "wss://fstream.binance.com"

[exchange.bybit]
enabled = true
ws_url = "wss://stream.bybit.com/v5/public/linear"

[storage.sqlite]
enabled = true
path = "data/xarb.db"

[arbitrage]
min_spread = 0.01  # Minimum 0.01% profit
```

## Benefits of DDD Architecture

✅ **Clear Separation**: Each layer has a single responsibility  
✅ **Testability**: Business logic independent of frameworks  
✅ **Extensibility**: Easy to add exchanges, storage, or strategies  
✅ **Maintainability**: Logical organization, clear dependencies  
✅ **Scalability**: Ready for complex features and optimizations  
✅ **Flexibility**: Swap implementations without changing logic  

## Key Files to Understand

1. **Entry Point**: `cmd/xarb/main.go`
2. **Container**: `internal/application/container/container.go`
3. **Monitor Use Case**: `internal/application/usecase/monitor/service.go`
4. **Core Logic**: `internal/domain/service/arbitrage_executor.go`
5. **Exchange Adapters**: `internal/infrastructure/exchange/{exchange}/`
