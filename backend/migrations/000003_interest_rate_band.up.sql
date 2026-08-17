ALTER TABLE mortgage_products
  ADD COLUMN interest_rate_min NUMERIC(6,3),
  ADD COLUMN interest_rate_max NUMERIC(6,3);

UPDATE mortgage_products
SET interest_rate_min = interest_rate,
    interest_rate_max = interest_rate
WHERE interest_rate IS NOT NULL
  AND interest_rate_min IS NULL
  AND interest_rate_max IS NULL;
