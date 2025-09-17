# MKY Trading Bot

A sophisticated cryptocurrency trading bot built in Go that implements automated trading strategies for the DOGEUSDT pair on Bybit exchange. The system uses technical analysis to identify breakout opportunities and automatically places trades when specific market conditions are met.

## 🎯 What Does This Bot Do?

This trading bot monitors DOGEUSDT price movements on a minute-by-minute basis and automatically executes trades when it detects certain market conditions:

- **Resistance/Support Breakouts**: When the system detects that price breaks above a resistance level (wall) or below a support level, it evaluates the trading opportunity
- **Volume Analysis**: The bot analyzes volume patterns to confirm the validity of breakouts
- **Automated Long/Short Positions**: When all conditions are met, the system automatically places either LONG or SHORT positions with predefined take-profit and stop-loss levels

## ⏰ Trading Schedule

The bot operates on an **hourly schedule**:
- Executes trading analysis every hour at the top of the hour (00 minutes)
- Monitors existing positions continuously
- Only places one order at a time to manage risk

## 🏗️ Project Status

**This is currently an MVP (Minimum Viable Product)** and is under active development. The project is subject to:
- Code refactoring and improvements
- Strategy optimizations
- Additional features and enhancements
- Bug fixes and stability improvements

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite (for local database)
- Bybit API credentials (for live trading)

### Environment Variables Setup

1. Copy the environment example file:
```bash
cp .env-example .env
```

2. Edit the `.env` file with your configuration:
```bash
# API Keys for exchanges
BINANCE_API_KEY=your_binance_api_key_here
BINANCE_SECRET_KEY=your_binance_secret_key_here

BYBIT_API_KEY=your_bybit_api_key_here
BYBIT_SECRET_KEY=your_bybit_secret_key_here

# General configurations
LOG_LEVEL=info
```

## Database Setup and Migrations

The bot uses SQLite for data storage and includes automated migration and initial data setup.

For now I'm using SQLite for semplicity, has future goal I want to move to postgres.

WIP: This parts needs to be refactored properly

#### 2. Populate Initial Data
The system automatically creates initial data when you run the bot for the first time, including:
- Order status entities (New, Filled, Cancelled, etc.)
- Database indexes for performance optimization
- Constraint triggers for data validation



## Running the Bot

### Option 1:

```bash
# Install dependencies
go mod tidy

# Run the bot
go run ./cmd/main.go
```

#### Option 2: Build and Run

```bash
# Build the executable
make build

# Run the built binary
./bin/trading-bot
```

## 📊 How the Trading Strategy Works

The bot implements a breakout trading strategy:

1. **Data Collection**: Every hour, the bot fetches the last 1000 1-minute candles for DOGEUSDT

2. **Technical Analysis**: 
   - Calculates resistance (wall) and support levels from the last 72 candles
   - Analyzes volume patterns for green (bullish) and red (bearish) candles

3. **Signal Detection**:
   - **LONG Signal**: When price closes above resistance with high volume confirmation
   - **SHORT Signal**: When price closes below support with high volume confirmation

4. **Position Management**:
   - Automatically sets take-profit levels (3% for LONG, 3% for SHORT)
   - Implements stop-loss protection (0.8% for both directions)
   - Uses available account balance to calculate position size

5. **Risk Management**:
   - Only one active position at a time
   - Monitors positions and updates database when trades are executed

## 🗄️ Database Schema

The bot maintains three main tables:
- `order_status_entities`: Order status definitions
- `orders`: Trading orders with full details
- `order_audits`: Audit trail for order changes


## ⚠️ Important Notes

1. **Production Trading**: Ensure you have sufficient funds and understand the risks before enabling live trading
2. **API Limits**: The bot respects exchange API rate limits
3. **Network Requirements**: Stable internet connection required for real-time trading
4. **Monitoring**: Monitor logs regularly for trading activity and system health

## 📝 Logs and Monitoring

The bot provides detailed logging including:
- Trading signal detection
- Order placement and status
- Volume analysis results
- Error handling and recovery

## 🔧 Configuration

Key configuration options can be modified in:
- Environment variables (`.env` file)
- Worker schedules in `worker/workers.go`
- Trading parameters in `worker/doge_trading_system.go`

## 🚨 Disclaimer

This trading bot is for educational and research purposes. Cryptocurrency trading involves significant risk, and you should only trade with funds you can afford to lose. Always test thoroughly before using with real funds.
