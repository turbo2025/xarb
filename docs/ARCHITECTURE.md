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

- **Ports** (`application/port/`) - Interface contracts
  - `PriceFeed` - Standard price feed interface with bidirectional symbol conversion
    - `Subscribe(ctx, coins)` - Subscribe to price updates
    - `Symbol2Coin(symbol)` - Convert exchange format to coin
    - `Coin2Symbol(coin)` - Convert coin to exchange format
  - `Repository` - Data persistence interface
  - `Sink` - Output abstraction (console, Feishu, etc.)

- **Services** (`application/service/`)
  - `ArbitrageCalculator` - Spread opportunity detection
  - `PriceService` - Price data management
  - `PositionService` - Position tracking
  - `SignalService` - Signal generation and filtering

- **Use Cases** (`application/usecase/monitor/`)
  - `service.go` - Main monitoring logic with Feishu integration
  - `state.go` - Price state machine and delta calculation
  - `formatter.go` - Display formatting with exchange filtering
  - `types.go` - Shared type definitions

**Characteristics:**
- Depends on `port` interfaces (dependency inversion)
- Relatively thin layer, mostly coordinating domain and infrastructure
- No business rules here, only process flow
- Easy to extend with new use cases (via Runnable interface)

### 3. Infrastructure Layer (`internal/infrastructure/`)

Technical implementations and external dependencies.

**Key Components:**

- **ServiceContext** (`svc/service_context.go`)
  - Central hub for all services and dependencies
  - Unified lifecycle management via `Runnable` interface
  - Single entry point for initialization and configuration
  - Handles graceful shutdown with `closerChain`
  - Concurrent service execution support

- **Exchange Integration** (`exchange/` & `factory/`)
  - Multiple exchange support: Binance, Bybit, OKX, Bitget
  - Unified symbol conversion via `SymbolConverter` interface
  - Spot and Perpetual market support
  - REST API clients for account, orders, positions
  - WebSocket clients for real-time price data

- **WebSocket Management** (`websocket/`)
  - Centralized connection management
  - Auto-reconnection with exponential backoff
  - Support for Spot and Perpetual WebSocket streams
  - Factory pattern for exchange-specific initializers

- **Storage Layer** (`storage/`)
  - SQLite - Default, embedded storage
  - Redis - Optional, for streaming and caching
  - PostgreSQL - Optional, for production deployments
  - Common interface: `Repository` and `ArbitrageRepository`

- **Configuration** (`config/`)
  - TOML-based configuration system
  - Coins + quote format for symbol management
  - Multi-storage backend support
  - Feishu notification configuration
  - Validation and default values

- **Logger** (`logger/`)
  - Structured logging with zerolog
  - Decoupled from business logic

### 4. Interfaces Layer (`internal/interfaces/`)

External-facing adapters and notification systems.

**Components:**

- **Console** (`console/`)
  - Real-time terminal output with ANSI colors
  - Live price updates with cursor control
  - Signal display with delta calculations

- **Feishu** (`feishu/`)
  - HTTP client for Feishu bot API
  - HMAC-SHA256 signature generation
  - Composite Sink combining console + Feishu output
  - Message formatting and ANSI stripping
  - Multi-channel support (signal, order, pnl, market)

## Data Flow

### Initialization Flow
```
main.go
    ↓
svc.New(cfg) → ServiceContext
    ↓
├── Load configuration (TOML)
├── Initialize exchange API clients
├── Set up WebSocket manager
│   └── Register price feeds from all exchanges
├── Initialize storage (SQLite/Redis/PostgreSQL)
├── Create Sink (Console or Feishu)
└── Instantiate Monitor Service
    ↓
RegisterService(monitorService) → runnableServices
    ↓
svc.Run(ctx) → Execute all services concurrently
```

### Price Update Flow
```
Exchange WebSocket
    ↓
ws_client.go (each exchange)
    ↓
Symbol2Coin() conversion (e.g., "BTCUSDT" → "BTC")
    ↓
Monitor Service goroutine
    ↓
State.Apply() - Update price state
    ↓
Formatter.Render() - Format display
    ↓
Sink.WriteLive() - Console or Feishu output
```

### Arbitrage Signal Flow
```
Price Update Event
    ↓
State.DeltaBand() - Calculate spread
    ↓
Band Threshold Crossing Detection
    ↓
Monitor Service: handleArbitrageSignal()
    ↓
├── Feishu Notification via Sink.SendSignal()
└── OrderManager.ExecuteArbitrage()
    ↓
Exchange API - Place orders
    ↓
verifyOrderExecution() - Confirm execution
```

### Symbol Conversion System
```
Configuration:
  coins = ["BTC", "ETH", "SOL"]
  quote = "USDT"
    ↓
Each Exchange Package:
  okx/ws_client.go
  bybit/ws_client.go
  binance/ws_client.go
  bitget/ws_client.go
    ↓
SymbolConverter Interface:
  Symbol2Coin(symbol string) string
  Coin2Symbol(coin string) string
    ↓
Package-level Singleton:
  symbolConverter SymbolConverter
    ↓
InitializeConverter(quote string) called once
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
- Symbol conversion adapters for each exchange
- Uniform interface (`PriceFeed`, `OrderClient`, etc.)
- Easy to add new exchanges without changing business logic

### 4. Factory Pattern
- Service context as central factory
- Exchange registry for dynamic initialization
- Runnable interface for pluggable services

### 5. Strategy Pattern
- Different trading strategies as implementations
- Signal detection via threshold crossing
- Easy to add new strategies

## Key Architecture Improvements

### Symbol Conversion System
- **Centralized**: Single `SymbolConverter` interface
- **Reusable**: Package-level singletons per exchange
- **Bidirectional**: `Symbol2Coin()` and `Coin2Symbol()` methods
- **Extensible**: Easy to support new symbol formats

### ServiceContext Pattern
- **Single Initialization Point**: Unified setup in `svc.New(cfg)`
- **Service Registry**: Support for multiple concurrent services via `Runnable` interface
- **Graceful Shutdown**: Automatic resource cleanup with `closerChain`
- **Dependency Management**: All services created and managed in one place

### Output Abstraction
- **Sink Interface**: Decouples business logic from output
- **Console Sink**: Real-time terminal output
- **Feishu Sink**: Composite sink supporting both console + Feishu notifications
- **ANSI Stripping**: Automatic cleanup of color codes for Feishu

## Project Structure

```
xarb/
├── cmd/
│   └── xarb/
│       └── main.go                 # Entry point, calls svc.New() and svc.Run()
├── configs/
│   └── config.toml                 # TOML configuration with coins + quote format
├── data/                           # Database files
├── docs/                           # Documentation
├── internal/
│   ├── application/
│   │   ├── port/                   # Interface contracts
│   │   │   ├── pricefeed.go        # Symbol2Coin/Coin2Symbol
│   │   │   ├── repository.go
│   │   │   └── sink.go             # Console/Feishu output
│   │   ├── service/                # Business orchestration
│   │   │   ├── arbitrage_calculator.go
│   │   │   ├── price_service.go
│   │   │   └── signal_service.go
│   │   └── usecase/                # Use case implementations
│   │       └── monitor/            # Price monitoring
│   │           ├── service.go      # Main logic with Feishu integration
│   │           ├── state.go        # Price state and delta calculation
│   │           └── formatter.go    # Display with ANSI formatting
│   ├── domain/
│   │   ├── model/                  # Domain models
│   │   │   ├── arbitrage.go
│   │   │   └── symbol.go
│   │   └── service/                # Core business logic
│   │       ├── arbitrage_executor.go
│   │       ├── order_manager.go
│   │       └── risk_manager.go
│   ├── infrastructure/
│   │   ├── config/                 # Configuration management
│   │   ├── exchange/               # Exchange adapters
│   │   │   ├── okx/
│   │   │   ├── bybit/
│   │   │   ├── binance/
│   │   │   ├── bitget/
│   │   │   └── symbol.go           # SymbolConverter implementations
│   │   ├── factory/                # API client factory
│   │   ├── svc/                    # ServiceContext and initialization
│   │   ├── storage/                # Data persistence
│   │   ├── websocket/              # WebSocket management
│   │   └── logger/                 # Structured logging
│   └── interfaces/
│       ├── console/                # Terminal output
│       └── feishu/                 # Feishu notifications
│           ├── client.go           # Feishu HTTP client
│           └── sink.go             # Composite Sink implementation
├── go.mod
├── go.sum
└── Makefile
```

## Extension Points

### Adding a New Exchange

1. Create new adapter in `infrastructure/exchange/{exchange}/`
2. Implement required interfaces:
   - `PriceFeed` - Real-time price streaming with `Symbol2Coin()` and `Coin2Symbol()`
   - `OrderClient` - Order placement and cancellation
   - `AccountClient` - Account information queries
3. Create `register.go` with `pricefeed.Register()` in `init()`
4. Implement `SymbolConverter` for exchange-specific format
5. Add configuration section in `config.toml`

### Adding New Storage Backend

1. Implement `repository.Repository` interface
2. Create in `infrastructure/storage/{backend}/`
3. Register in `ServiceContext.initializeStorage()`
4. Add configuration option in `config.toml`

### Adding New Service/Use Case

1. Create service implementing `Runnable` interface
2. Initialize in `ServiceContext`
3. Register via `RegisterService(service)`
4. Runs concurrently with other services

## Testing Strategy

- **Domain Services**: Pure unit tests, no framework dependencies
- **Application Services**: Mock infrastructure via `port` interfaces
- **Infrastructure**: Integration tests with test databases
- **Use Cases**: End-to-end tests with mock services
- **Monitor Service**: Test with mock price feeds and sinks

## Configuration

Edit `configs/config.toml`:

```toml
[app]
print_every_min = 5

[symbols]
coins = ["BTC", "ETH", "SOL"]
quote = "USDT"

[arbitrage]
delta_threshold = 5.0  # 5% spread threshold

[monitor]
exchanges = ["binance", "bybit", "okx"]  # Optional: specify exchanges to monitor

[exchanges.binance]
enabled = true
api_key = "your_key"
secret_key = "your_secret"
perpetual_http_url = "https://fapi.binance.com"
perpetual_ws_url = "wss://fstream.binance.com/ws"

[exchanges.bybit]
enabled = true
api_key = "your_key"
secret_key = "your_secret"
perpetual_http_url = "https://api.bybit.com"
perpetual_ws_url = "wss://stream.bybit.com/v5/public/linear"

[exchanges.okx]
enabled = true
api_key = "your_key"
secret_key = "your_secret"
passphrase = "your_passphrase"
perpetual_http_url = "https://www.okx.com"
perpetual_ws_url = "wss://ws.okx.com:8443/ws/v5/public"

[message.feishu]
[[message.feishu]]
channel = "signal"
webhook = "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
secret = "your_secret_key"

[sqlite]
enabled = true
path = "data/xarb.db"

[redis]
enabled = false
addr = "127.0.0.1:6379"
password = ""
db = 0
```

## System Startup Sequence

1. **main.go**
   - Parse command-line flags
   - Initialize logger
   - Load configuration from `config.toml`

2. **ServiceContext.New(cfg)**
   - Initialize exchange API clients
   - Create WebSocket manager
   - Register price feeds from all enabled exchanges
   - Initialize storage layer
   - Create Sink (Console or Feishu)
   - Instantiate Monitor Service
   - Add services to `runnableServices`

3. **ServiceContext.Run(ctx)**
   - Execute all registered services concurrently
   - Each service runs independently
   - First service error returns immediately
   - Remaining services cleaned up via `ctx.Done()`

4. **Graceful Shutdown**
   - SIGINT/SIGTERM handling in main
   - Cancel context
   - Close all running services
   - Execute cleanup chain (`closerChain`)

## Benefits of This Architecture

✅ **Clear Separation**: Each layer has single responsibility  
✅ **Testability**: Business logic independent of frameworks  
✅ **Extensibility**: Easy to add exchanges, storage, or services  
✅ **Maintainability**: Logical organization, clear dependencies  
✅ **Scalability**: Ready for complex features and optimizations  
✅ **Flexibility**: Swap implementations without changing logic  
✅ **Concurrency**: Multiple services running safely in parallel  
✅ **Notifications**: Built-in Feishu integration for real-time alerts  

## Key Files to Start With

1. **Entry Point**: [cmd/xarb/main.go](../../cmd/xarb/main.go)
2. **Service Hub**: [internal/infrastructure/svc/service_context.go](../../internal/infrastructure/svc/service_context.go)
3. **Monitor Logic**: [internal/application/usecase/monitor/service.go](../../internal/application/usecase/monitor/service.go)
4. **Core Business**: [internal/domain/service/arbitrage_executor.go](../../internal/domain/service/arbitrage_executor.go)
5. **Exchange Adapters**: [internal/infrastructure/exchange/okx/ws_client.go](../../internal/infrastructure/exchange/okx/ws_client.go)
```
