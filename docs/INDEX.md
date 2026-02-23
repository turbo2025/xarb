# XARB Documentation Index

Complete guide to XARB - Multi-exchange cryptocurrency arbitrage bot with real-time monitoring and Feishu notifications.

## 📚 Core Documentation

### 1. **[README.md](readme.md)** - Project Overview
Start here for:
- Feature overview
- Quick start guide
- Installation and configuration
- Symbol management concepts
- Arbitrage detection flow

**Best for**: First-time users, getting started, understanding what XARB does.

### 2. **[ARCHITECTURE.md](ARCHITECTURE.md)** - System Design
Complete technical architecture:
- DDD (Domain-Driven Design) structure
- Layer-by-layer breakdown (Domain, Application, Infrastructure, Interfaces)
- Data flow diagrams
- Symbol conversion system
- ServiceContext pattern
- Extension points and testing strategy

**Best for**: Developers contributing to the codebase, understanding design patterns, adding new features.

### 3. **[ARBITRAGE.md](ARBITRAGE.md)** - Trading Strategies
Arbitrage mechanism details:
- Trading pairs and profitability calculation
- Three arbitrage strategies (spread, funding, triangular)
- Risk assessment and management
- Real-world examples with numbers

**Best for**: Understanding arbitrage mechanics, calculating profitability, risk analysis.

### 4. **[WEBSOCKET_ARCHITECTURE.md](WEBSOCKET_ARCHITECTURE.md)** - Data Streaming Design
WebSocket and real-time data:
- Manager architecture
- Distributed factory registration system
- PriceFeed and OrderBook interfaces
- Exchange-specific implementations
- Extensibility for new data sources

**Best for**: Understanding real-time data flow, adding WebSocket support, data source design.

## 🎯 Quick Navigation

### By Role

**🚀 Getting Started**
- Read [README.md](readme.md#quick-start) for installation
- Configure `configs/config.toml` with your exchanges
- Run `go build ./cmd/xarb`

**👨‍💻 Developers**
- Understand [ARCHITECTURE.md](ARCHITECTURE.md) - system design
- Review [WEBSOCKET_ARCHITECTURE.md](WEBSOCKET_ARCHITECTURE.md) - data sources
- Check code comments in `internal/` for implementation details

**📊 Traders/Operators**
- Configure exchanges in [README.md](readme.md#configuration)
- Set up Feishu notifications
- Understand spreads in [ARBITRAGE.md](ARBITRAGE.md)
- Monitor alerts in console output

**🔧 Contributors**
- Study [ARCHITECTURE.md](ARCHITECTURE.md#extension-points) for extension patterns
- Review [ARCHITECTURE.md](ARCHITECTURE.md#testing-strategy) for testing approach
- Run tests: `go test ./...`

### By Topic

**💰 Arbitrage & Trading**
- [ARBITRAGE.md](ARBITRAGE.md) - Complete strategies

**🏗️ System Design**
- [ARCHITECTURE.md](ARCHITECTURE.md) - Full DDD structure
- [WEBSOCKET_ARCHITECTURE.md](WEBSOCKET_ARCHITECTURE.md) - Real-time data

**⚙️ Configuration**
- [README.md](readme.md#configuration) - All config options
- [ARBITRAGE.md](ARBITRAGE.md) - Strategy parameters

**🔌 Integration**
- [WEBSOCKET_ARCHITECTURE.md](WEBSOCKET_ARCHITECTURE.md#扩展指南) - Adding exchanges
- [ARCHITECTURE.md](ARCHITECTURE.md#extension-points) - Adding services

**📞 Notifications**
- [README.md](readme.md#feishu-notifications) - Setup Feishu bot

## 📖 Document Status

| Document | Status | Last Updated | Coverage |
|----------|--------|--------------|----------|
| [README.md](readme.md) | ✅ Current | Feb 23, 2026 | Features, setup, config |
| [ARCHITECTURE.md](ARCHITECTURE.md) | ✅ Current | Feb 23, 2026 | Full system design |
| [ARBITRAGE.md](ARBITRAGE.md) | ✅ Current | Feb 23, 2026 | Trading strategies |
| [WEBSOCKET_ARCHITECTURE.md](WEBSOCKET_ARCHITECTURE.md) | ✅ Current | Feb 23, 2026 | Data streaming |

## 🚀 Getting Started Path

1. **Understand the project** (5 min)
   - Read [README.md](readme.md) overview

2. **Set up locally** (10 min)
   - Follow [Quick Start](readme.md#quick-start)
   - Configure exchanges in `config.toml`

3. **Run and monitor** (5 min)
   - Start XARB: `go build ./cmd/xarb && ./xarb`
   - Watch console for price updates and signals

4. **Understand the code** (30 min)
   - Study [ARCHITECTURE.md](ARCHITECTURE.md) overview
   - Check `cmd/xarb/main.go` entry point
   - Review `internal/infrastructure/svc/service_context.go`

5. **Learn strategies** (15 min)
   - Read [ARBITRAGE.md](ARBITRAGE.md)
   - Understand spread calculation

6. **Contribute or extend** (varies)
   - Review [ARCHITECTURE.md#extension-points](ARCHITECTURE.md#extension-points)
   - Follow [Testing Strategy](ARCHITECTURE.md#testing-strategy)

## 💡 Key Concepts

### Symbol Management
- **Config format**: `coins = ["BTC", "ETH"]` + `quote = "USDT"`
- **Conversion**: Each exchange converts to its format (Binance: `BTCUSDT`, OKX: `BTC-USDT-SWAP`)
- **Flow**: WebSocket Symbol → Symbol2Coin → State → Display

### Service Architecture
- **ServiceContext** - Central hub for all services
- **Runnable interface** - For concurrent services
- **Sink abstraction** - Output to console or Feishu

### Data Flow
- WebSocket feeds → Symbol conversion → State updates → Spread detection → Notifications/Execution

## 🔗 Related Files

Key code files mentioned in docs:

- `cmd/xarb/main.go` - Application entry point
- `internal/infrastructure/svc/service_context.go` - Service hub
- `internal/application/usecase/monitor/service.go` - Main monitoring logic
- `internal/domain/service/arbitrage_executor.go` - Core business logic
- `internal/infrastructure/exchange/*/ws_client.go` - Exchange adapters
- `internal/infrastructure/config/config.go` - Configuration
- `internal/interfaces/feishu/client.go` - Feishu notifications

## ❓ FAQ

**Q: How do I add a new exchange?**  
A: See [ARCHITECTURE.md#adding-a-new-exchange](ARCHITECTURE.md#extension-points)

**Q: How do I customize the monitoring?**  
A: Edit `configs/config.toml` and review [README.md#configuration](readme.md#configuration)

**Q: How are prices converted between exchanges?**  
A: See [ARCHITECTURE.md#symbol-conversion-system](ARCHITECTURE.md#data-flow)

**Q: Can I use Feishu notifications in my own code?**  
A: Yes, see [interfaces/feishu/](../internal/interfaces/feishu/) for the client API

**Q: How do I run tests?**  
A: `go test ./...` - see [ARCHITECTURE.md#testing-strategy](ARCHITECTURE.md#testing-strategy)

## 📝 Notes

- All documentation is kept in sync with code changes
- Architecture follows Domain-Driven Design (DDD)
- Configuration uses TOML format
- Main language is Go 1.21+

---

**Last Updated**: February 23, 2026  
**Status**: ✅ All documents current and tested
