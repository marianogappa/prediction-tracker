# Predictions Tracker

A local-first, event-driven prediction tracking engine that ingests crypto price predictions, evaluates them deterministically against live market data, and produces trustable, exportable results.

## Quick Start

```bash
make build
./bin/predictions-server
```

Open http://localhost:8080 to view the UI.

## Architecture

- **Domain Model**: Predictions are deterministic state machines (draft → enabled → monitoring → final_*)
- **Event-Driven**: All state changes emit immutable, timestamped events; state is reconstructable from events
- **Evaluation Engine**: Deterministic Go code evaluates threshold, crossing, and sustained-duration rules
- **Source of Truth**: [crypto-candles](https://github.com/marianogappa/crypto-candles) provides OHLC data from major exchanges
- **Storage**: SQLite (behind an interface) with WAL mode for concurrent reads

## Configuration

| Env Var | Flag | Default | Description |
|---------|------|---------|-------------|
| `PREDICTIONS_DB_PATH` | `-db` | `./predictions.db` | SQLite database path |
| `PREDICTIONS_POLL_INTERVAL` | `-poll-interval` | `60s` | Polling interval for price data |
| `PREDICTIONS_LISTEN_ADDR` | `-listen` | `:8080` | HTTP listen address |

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/predictions` | List predictions (filter by `?state=`) |
| GET | `/api/predictions/{id}` | Get prediction detail |
| POST | `/api/predictions` | Create prediction (draft) |
| POST | `/api/predictions/{id}/enable` | Enable prediction |
| POST | `/api/predictions/{id}/disable` | Disable prediction |
| GET | `/api/predictions/{id}/events` | List events |
| GET | `/api/predictions/{id}/values` | List candle values |
| GET | `/api/predictions/{id}/export` | Download markdown export |

## LLM Ingestion

```bash
export OPENAI_API_KEY=sk-...
./bin/predictions-ingest "I think BTC will hit 100k by end of 2026"
```

## Creating a Prediction via API

```bash
curl -X POST http://localhost:8080/api/predictions \
  -H "Content-Type: application/json" \
  -d '{
    "statement": "BTC will reach $100,000 by end of 2026",
    "rule": {
      "type": "threshold",
      "operator": ">=",
      "price_field": "close",
      "value": 100000,
      "deadline": "2026-12-31T00:00:00Z"
    },
    "asset": "BINANCE:BTC/USDT",
    "deadline": "2026-12-31T00:00:00Z",
    "author_name": "Alice",
    "source_url": "https://example.com/prediction"
  }'
```

Then enable it:

```bash
curl -X POST http://localhost:8080/api/predictions/{id}/enable
```

## Testing

```bash
make test
```

## Rule Types

**threshold**: Price meets condition before deadline.
```json
{"type": "threshold", "operator": ">=", "price_field": "close", "value": 100000, "deadline": "2026-12-31T00:00:00Z"}
```

**crossing**: Price crosses a target value (from below to above or vice versa).
```json
{"type": "crossing", "operator": ">=", "price_field": "close", "value": 100000, "deadline": "2026-12-31T00:00:00Z"}
```

**sustained_duration**: Price meets condition continuously for a duration.
```json
{"type": "sustained_duration", "operator": ">=", "price_field": "close", "value": 100000, "duration_ms": 3600000, "deadline": "2026-12-31T00:00:00Z"}
```
