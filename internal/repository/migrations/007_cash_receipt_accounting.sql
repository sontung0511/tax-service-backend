CREATE TABLE IF NOT EXISTS cash_receipt_invoices (
    id TEXT PRIMARY KEY,
    voucher_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    invoice_no TEXT NOT NULL,
    invoice_symbol TEXT NOT NULL DEFAULT '',
    invoice_date DATE,
    tax_code TEXT NOT NULL DEFAULT '',
    UNIQUE (voucher_id, id)
);

CREATE INDEX IF NOT EXISTS idx_cash_receipt_invoices_voucher ON cash_receipt_invoices(voucher_id);

CREATE TABLE IF NOT EXISTS cash_receipt_revenue_details (
    id TEXT PRIMARY KEY,
    voucher_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    invoice_id TEXT NOT NULL REFERENCES cash_receipt_invoices(id) ON DELETE CASCADE,
    debit_account VARCHAR(20) NOT NULL REFERENCES accounts(code),
    credit_account VARCHAR(20) NOT NULL REFERENCES accounts(code),
    detail_code TEXT NOT NULL,
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(20,2) NOT NULL CHECK (unit_price >= 0),
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL,
    UNIQUE (voucher_id, id)
);

CREATE INDEX IF NOT EXISTS idx_cash_receipt_revenue_invoice ON cash_receipt_revenue_details(invoice_id);

CREATE TABLE IF NOT EXISTS cash_receipt_tax_lines (
    id TEXT PRIMARY KEY,
    voucher_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    invoice_id TEXT NOT NULL REFERENCES cash_receipt_invoices(id) ON DELETE CASCADE,
    revenue_detail_id TEXT NOT NULL REFERENCES cash_receipt_revenue_details(id) ON DELETE CASCADE,
    tax_rate NUMERIC(5,2) NOT NULL CHECK (tax_rate IN (0, 5, 8, 10)),
    taxable_amount NUMERIC(20,2) NOT NULL CHECK (taxable_amount >= 0),
    tax_amount NUMERIC(20,2) NOT NULL CHECK (tax_amount >= 0),
    price_includes_tax BOOLEAN NOT NULL,
    UNIQUE (voucher_id, revenue_detail_id)
);

CREATE INDEX IF NOT EXISTS idx_cash_receipt_tax_invoice ON cash_receipt_tax_lines(invoice_id);
