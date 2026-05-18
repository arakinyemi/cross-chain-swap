CREATE TABLE swaps (
    id               TEXT PRIMARY KEY,
    from_chain       TEXT NOT NULL,
    to_chain         TEXT NOT NULL,
    from_asset       TEXT NOT NULL DEFAULT 'USDT',
    to_asset         TEXT NOT NULL DEFAULT 'USDT',
    amount_in        NUMERIC NOT NULL,
    amount_out       NUMERIC NOT NULL,
    fee              NUMERIC NOT NULL,
    deposit_address  TEXT NOT NULL UNIQUE,
    dest_address     TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'awaiting_deposit',
    btc_amount       TEXT,
    trade1_order_id  TEXT,
    trade2_order_id  TEXT,
    withdrawal_tx_id TEXT,
    reference        TEXT UNIQUE NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_swaps_deposit_address ON swaps(deposit_address);
CREATE INDEX idx_swaps_reference ON swaps(reference);
CREATE INDEX idx_swaps_status ON swaps(status);
