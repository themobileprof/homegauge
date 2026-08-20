-- Pre-disbursement Fund & settle (Paystack DVA collections)

CREATE TABLE IF NOT EXISTS funding_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL UNIQUE REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  paystack_customer_code TEXT NOT NULL DEFAULT '',
  paystack_dva_id BIGINT,
  account_number TEXT NOT NULL DEFAULT '',
  account_name TEXT NOT NULL DEFAULT '',
  bank_name TEXT NOT NULL DEFAULT '',
  bank_slug TEXT NOT NULL DEFAULT '',
  currency_code TEXT NOT NULL DEFAULT 'NGN',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS funding_obligations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  obligation_key TEXT NOT NULL,
  label TEXT NOT NULL,
  amount NUMERIC(18,2),
  amount_received NUMERIC(18,2) NOT NULL DEFAULT 0,
  currency_code TEXT NOT NULL DEFAULT 'NGN',
  due_phase TEXT NOT NULL DEFAULT 'before_approval',
  collectable BOOLEAN NOT NULL DEFAULT TRUE,
  status TEXT NOT NULL DEFAULT 'pending',
  note TEXT NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (application_id, obligation_key)
);

CREATE TABLE IF NOT EXISTS funding_movements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  direction TEXT NOT NULL DEFAULT 'credit',
  amount NUMERIC(18,2) NOT NULL,
  currency_code TEXT NOT NULL DEFAULT 'NGN',
  paystack_reference TEXT NOT NULL DEFAULT '',
  paystack_event TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'success',
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (paystack_reference)
);

CREATE INDEX IF NOT EXISTS idx_funding_obligations_app ON funding_obligations(application_id);
CREATE INDEX IF NOT EXISTS idx_funding_movements_app ON funding_movements(application_id);
CREATE INDEX IF NOT EXISTS idx_funding_accounts_user ON funding_accounts(user_id);
