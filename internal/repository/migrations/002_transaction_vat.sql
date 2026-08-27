ALTER TABLE transactions
    ADD COLUMN
IF NOT EXISTS vat_amount BIGINT NOT NULL DEFAULT 0 CHECK
(vat_amount >= 0);
