ALTER TABLE users
  ADD COLUMN lender_id UUID REFERENCES lenders(id);

CREATE INDEX users_lender_idx ON users (lender_id)
  WHERE deleted_at IS NULL AND role = 'LENDER_USER';

COMMENT ON COLUMN users.lender_id IS 'Set for LENDER_USER accounts. Lenders without a linked user are updated by advisors.';
