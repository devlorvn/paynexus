-- 1. Tạo bảng merchants với cột resource_path chuẩn Multi-Tenancy
CREATE TABLE IF NOT EXISTS public.merchants (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    resource_path VARCHAR(255) NOT NULL DEFAULT '/org/default/',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Kích hoạt Row Level Security (RLS) trên bảng merchants
ALTER TABLE public.merchants ENABLE ROW LEVEL SECURITY;

-- 3. Tạo RLS Policy tự động lọc theo session variable 'permission.resource_path'
DROP POLICY IF EXISTS merchant_rls_policy ON public.merchants;
CREATE POLICY merchant_rls_policy ON public.merchants
    USING (resource_path = current_setting('permission.resource_path', true));
