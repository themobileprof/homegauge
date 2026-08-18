DROP INDEX IF EXISTS users_lender_idx;
ALTER TABLE users DROP COLUMN IF EXISTS lender_id;
