ALTER TABLE mortgage_products
  DROP COLUMN IF EXISTS interest_rate_min,
  DROP COLUMN IF EXISTS interest_rate_max;
