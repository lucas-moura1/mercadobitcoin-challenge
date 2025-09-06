# MercadoBitcoin Challenge (Go)

A minimal matching engine and trading API in Go. It exposes endpoints to create and cancel orders, retrieve an aggregated order book, and fetch account balances. Business logic is covered by table-driven tests with gomock and testify/assert.

## Requirements

- Go 1.21+ (recommended 1.25)
- Git
- Docker (for running with Postgres)

## Project Layout (high level)

- `entity/`: domain entities (Order, Trade, Wallet, etc.)
- `repository/`: data access interfaces and implementations
- `usecase/`: core business logic (order creation, matching, trade execution)
- `handler/`: HTTP handlers
- `cmd/<app>` or root `main.go`: application entrypoint
- `Tests`: table-driven, with gomock-based repository/use case mocks


## Run

- Clone the repo:
```
git clone https://github.com/yourusername/mercadobitcoin-challenge.git
cd mercadobitcoin-challenge
```
Run with Docker Compose (Postgres + app):
```
docker compose up
```

Server listens on `8080`.

### Database setup and seeding

1) Start services
```
docker compose up -d
```

2) Apply schema.sql (pick one)

Example (adjust to your compose):
```
docker compose exec -T db psql -U postgres -d clob_db < ./scripts/schema.sql
```

3) Run the seeder inside the app container
```
docker compose exec service go run ./scripts/seed.go
```


## API

- POST `/orders`: Create an order
  - Request:
    ```
    {
      "account_id": "3f2b9f9c-0c57-4b2a-9e3a-0a3f6e8c7c11",
      "instrument_pair": "BTC_BRL",
      "order_type": "BUY",            // or "SELL"
      "price": "200000.00",
      "quantity": "0.50"
    }
    ```
  - Responses:
    - 201 Created:
      ```
      {
        "order_id": "…",
        "instrument_pair": "BTC_BRL",
        "order_type": "BUY",
        "price": "200000.00",
        "quantity": "0.50",
        "status": "OPEN"
      }
      ```
    - 400 on validation/business errors

- POST `/orders/{id}/cancel`: Cancel an order
  - Responses: 200 on success; 400/404/500 on error

- GET `/orderbook/{instrument_pair}`: Aggregated order book
  - `instrument_pair` format: `BASE_QUOTE` (e.g., `BTC_BRL`)
  - 200 OK:
    ```
    {
      "instrument_pair": "BTC_BRL",
      "bids": [ { "price": "100", "quantity": "1.4" }, … ],
      "asks": [ { "price": "101", "quantity": "0.8" }, … ]
    }
    ```
  - 404 if no open orders

- GET `/accounts/{id}/balance`: Account balances
  - 200 OK:
    ```
    [
      { "asset_symbol": "BTC", "balance": "0.5" },
      { "asset_symbol": "BRL", "balance": "1000" }
    ]
    ```
  - 404 if account has no wallets

## Verify It Works

Quickest verification is via tests (covers matching, settlement, order book aggregation, and handlers):

- Run all tests:
  - `go test ./...`

Key test areas:
- `usecase/order_usecase_test.go`: order book aggregation and CreateOrder
- `usecase/trade_executor_test.go`: Execute, settle, and status updates
- `handler/*_test.go`: handlers (CreateOrder, CancelOrder, GetOrderBook, GetAccountBalance)

API manual checks (requires seeded data):
1. Ensure wallets exist with sufficient balances for the test accounts (BRL for BUY, base asset for SELL).
2. Create a BUY and a SELL at matching prices.
3. Check `/orders/BTC_BRL` to see aggregated levels.
4. Check `/accounts/{id}/balance` to observe updated balances after matches.

Note: This project does not expose wallet funding endpoints; seed via migrations/fixtures or direct DB inserts during local testing.

## Implementation Details and Design Decisions

- Decimal arithmetic: uses `shopspring/decimal` for price/quantity to avoid float issues.
- Instrument pair format: `BASE_QUOTE` (validated; e.g., `BTC_BRL`).
- Order statuses: `OPEN`, `PARTIALLY_FILLED`, `FILLED`, `CANCELLED`.
- Order book: aggregated by price level (sum of `RemainingQuantity` per price), then sorted:
  - Bids: price descending
  - Asks: price ascending
  - Keys use canonical decimal strings to avoid duplicate levels like `100` vs `100.0`.
- Matching logic:
  - Matching Order vs. Order semantics; price taken from the matching Order.
  - Executes trades in order of best price, stops when taker is fully filled.
  - Settlement transfers base from seller→buyer and quote from buyer→seller.
- Transactions: GORM-based; use cases operate within a transaction boundary to ensure atomicity.
- Testing strategy:
  - Table-driven tests for all use cases and handlers.
  - gomock for repositories/use cases; assertions with testify/assert.
  - SQLite in-memory for obtaining a concrete `*gorm.DB` when needed in tests.
  - Gomock-generated mocks for interfaces in `repository` and `usecase`.

## Assumptions

- Supported assets include `BTC` and `BRL` in examples; any `BASE_QUOTE` pair conforming to the format is accepted.
- Wallets must pre-exist with sufficient balances; CreateOrder checks coverage.
- Matching is price-time within the constraints of repository return order.
