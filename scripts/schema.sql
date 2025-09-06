CREATE EXTENSION
IF NOT EXISTS "uuid-ossp";

CREATE TABLE account
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL
);

CREATE TABLE wallet
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL,
    asset_symbol VARCHAR(10) NOT NULL,
    balance DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES account(id),
    UNIQUE (account_id, asset_symbol),
    deleted_at TIMESTAMP NULL DEFAULT NULL
);

CREATE TABLE "order"
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL,
    instrument_pair VARCHAR(20) NOT NULL,
    order_type VARCHAR(4) NOT NULL CHECK (order_type IN ('BUY', 'SELL')),
    price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    remaining_quantity DECIMAL(20,8) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('OPEN', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES account(id)
);

CREATE TABLE trade
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    buyer_order_id UUID NOT NULL,
    seller_order_id UUID NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    FOREIGN KEY (buyer_order_id) REFERENCES "order"(id),
    FOREIGN KEY (seller_order_id) REFERENCES "order"(id)
);

-- Indexes
CREATE INDEX idx_wallet_account_id ON wallet(account_id);
CREATE INDEX idx_order_account_id_instrument_pair ON "order"(account_id, instrument_pair);
CREATE INDEX idx_order_match
  ON "order" (instrument_pair, order_type, price, created_at)
  WHERE status IN ('OPEN','PARTIALLY_FILLED');
