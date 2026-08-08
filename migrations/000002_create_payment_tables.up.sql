-- 1. Bảng Invoices (Hóa đơn nạp tiền)
CREATE TABLE IF NOT EXISTS public.invoices (
    id BIGSERIAL PRIMARY KEY,
    merchant_id BIGINT NOT NULL,
    invoice_code VARCHAR(64) NOT NULL UNIQUE,
    order_reference VARCHAR(128) NOT NULL,
    currency VARCHAR(16) NOT NULL DEFAULT 'USDT',
    amount NUMERIC(36, 18) NOT NULL,
    deposit_address VARCHAR(128) NOT NULL,
    memo_code VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- PENDING, SETTLED, EXPIRED
    resource_path VARCHAR(255) NOT NULL DEFAULT '/org/default/',
    expired_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Kích hoạt RLS cho invoices
ALTER TABLE public.invoices ENABLE ROW LEVEL SECURITY;
CREATE POLICY invoice_rls_policy ON public.invoices
    USING (resource_path = current_setting('permission.resource_path', true));

-- 2. Bảng Balances (Số dư tài khoản Merchant)
CREATE TABLE IF NOT EXISTS public.balances (
    id BIGSERIAL PRIMARY KEY,
    merchant_id BIGINT NOT NULL,
    currency VARCHAR(16) NOT NULL,
    available_balance NUMERIC(36, 18) NOT NULL DEFAULT 0,
    locked_balance NUMERIC(36, 18) NOT NULL DEFAULT 0,
    resource_path VARCHAR(255) NOT NULL DEFAULT '/org/default/',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(merchant_id, currency)
);

-- Kích hoạt RLS cho balances
ALTER TABLE public.balances ENABLE ROW LEVEL SECURITY;
CREATE POLICY balance_rls_policy ON public.balances
    USING (resource_path = current_setting('permission.resource_path', true));

-- 3. Bảng Ledger Entries (Sổ cái ghi đúp bất biến - Append Only)
CREATE TABLE IF NOT EXISTS public.ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(64) NOT NULL, -- Mã giao dịch tổng
    merchant_id BIGINT NOT NULL,
    account_type VARCHAR(32) NOT NULL,   -- ASSET, LIABILITY, REVENUE, EXPENSE
    entry_type VARCHAR(16) NOT NULL,     -- DEBIT hoặc CREDIT
    currency VARCHAR(16) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    resource_path VARCHAR(255) NOT NULL DEFAULT '/org/default/',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Kích hoạt RLS cho ledger_entries
ALTER TABLE public.ledger_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY ledger_rls_policy ON public.ledger_entries
    USING (resource_path = current_setting('permission.resource_path', true));
