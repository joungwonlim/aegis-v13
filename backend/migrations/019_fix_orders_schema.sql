-- Migration: 019_fix_orders_schema
-- Description: Add missing columns to execution.orders for repository compatibility
-- Date: 2026-01-10

-- Add order_id column (unique identifier for the order)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS order_id VARCHAR(50) UNIQUE;

-- Add code alias (maps to stock_code)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS code VARCHAR(20);

-- Add name alias (maps to stock_name)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS name VARCHAR(200);

-- Add side column (BUY/SELL instead of order_action)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS side VARCHAR(10);

-- Add quantity column (alias for order_qty)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS quantity INTEGER;

-- Add price column (alias for order_price)
ALTER TABLE execution.orders
ADD COLUMN IF NOT EXISTS price NUMERIC(12,2);

-- Update existing data
UPDATE execution.orders SET
    code = stock_code,
    name = stock_name,
    side = order_action,
    quantity = order_qty,
    price = order_price
WHERE code IS NULL;

-- Create index for order_id
CREATE INDEX IF NOT EXISTS idx_orders_order_id ON execution.orders(order_id);

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 019: execution.orders schema fixed successfully';
END $$;
