CREATE TABLE IF NOT EXISTS business_profiles (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    taxpayer_type TEXT NOT NULL CHECK (taxpayer_type IN ('household', 'enterprise')),
    tax_code TEXT NOT NULL,
    business_name TEXT NOT NULL,
    owner_name TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    industry TEXT NOT NULL,
    declaration_kind TEXT NOT NULL CHECK (declaration_kind IN ('month', 'quarter')),
    enterprise_type TEXT,
    legal_representative TEXT,
    vat_method TEXT,
    accounting_regime TEXT,
    household_tax_method TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tax_periods (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    year INTEGER NOT NULL CHECK (year BETWEEN 2000 AND 2200),
    kind TEXT NOT NULL CHECK (kind IN ('month', 'quarter')),
    status TEXT NOT NULL CHECK (status IN ('draft', 'review', 'completed')),
    due_date DATE NOT NULL,
    paid_amount BIGINT NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    locked_at TIMESTAMPTZ,
    tax_snapshot JSONB
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    period_id TEXT NOT NULL REFERENCES tax_periods(id) ON DELETE RESTRICT,
    transaction_date DATE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('revenue', 'expense')),
    description TEXT NOT NULL,
    invoice_no TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    revenue_category TEXT NOT NULL,
    document_no TEXT,
    payment_status TEXT CHECK (payment_status IS NULL OR payment_status IN ('paid', 'unpaid')),
    outstanding_amount BIGINT NOT NULL DEFAULT 0 CHECK (outstanding_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_period_date ON transactions(period_id, transaction_date);
CREATE UNIQUE INDEX IF NOT EXISTS uq_transactions_invoice ON transactions(transaction_date, LOWER(BTRIM(invoice_no)), amount);

CREATE TABLE IF NOT EXISTS tax_declarations (
    id TEXT PRIMARY KEY,
    tax_type TEXT NOT NULL CHECK (tax_type IN ('vat', 'pit', 'cit')),
    form_code TEXT NOT NULL,
    form_name TEXT NOT NULL,
    period_label TEXT NOT NULL,
    period_type TEXT NOT NULL CHECK (period_type IN ('month', 'quarter', 'year')),
    status TEXT NOT NULL CHECK (status IN ('draft', 'valid', 'locked')),
    schema_version TEXT NOT NULL,
    values JSONB NOT NULL DEFAULT '{}'::JSONB,
    updated_at TIMESTAMPTZ NOT NULL,
    locked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS audit_entries (
    id TEXT PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL,
    action TEXT NOT NULL,
    detail TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_entries_occurred_at ON audit_entries(occurred_at DESC);
