CREATE TABLE IF NOT EXISTS counterparties (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tax_code TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS voucher_type TEXT,
    ADD COLUMN IF NOT EXISTS counterparty_code TEXT,
    ADD COLUMN IF NOT EXISTS counterparty_name TEXT,
    ADD COLUMN IF NOT EXISTS counterparty_tax_code TEXT,
    ADD COLUMN IF NOT EXISTS counterparty_address TEXT;

CREATE INDEX IF NOT EXISTS idx_counterparties_code ON counterparties (code);
