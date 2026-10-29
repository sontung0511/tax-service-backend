ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check CHECK (type IN ('revenue', 'expense', 'carry_forward'));
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS cash_payment_data JSONB;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS carry_forward_data JSONB;
