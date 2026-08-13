CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('CUSTOMER', 'ADVISOR', 'ADMIN', 'LENDER_USER');
CREATE TYPE verification_status AS ENUM ('verified', 'needs_verification', 'expired');
CREATE TYPE eligibility_outcome AS ENUM (
  'likely_eligible',
  'potentially_eligible',
  'may_require_review',
  'unlikely',
  'more_info_required'
);
CREATE TYPE document_status AS ENUM (
  'not_started',
  'uploaded',
  'under_review',
  'accepted',
  'rejected',
  'requires_replacement'
);
CREATE TYPE application_status AS ENUM (
  'NEW',
  'ASSESSMENT_COMPLETED',
  'DOCUMENTS_PENDING',
  'DOCUMENTS_UNDER_REVIEW',
  'READY_FOR_SUBMISSION',
  'SUBMITTED_TO_LENDER',
  'LENDER_REVIEW',
  'ADDITIONAL_INFORMATION_REQUIRED',
  'APPROVED',
  'REJECTED',
  'COMPLETED',
  'CANCELLED'
);
CREATE TYPE automation_level AS ENUM ('suggest_only', 'auto_safe', 'auto_aggressive');

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'CUSTOMER',
  email_verified_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX users_email_unique ON users (LOWER(email)) WHERE deleted_at IS NULL;

CREATE TABLE user_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  full_name TEXT NOT NULL DEFAULT '',
  date_of_birth DATE,
  state_of_residence TEXT,
  residency_type TEXT,
  marital_status TEXT,
  phone TEXT,
  is_diaspora BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE employment_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  employment_type TEXT NOT NULL DEFAULT 'salaried',
  employer_name TEXT,
  years_employed NUMERIC(4,1),
  monthly_net_income NUMERIC(14,2),
  other_monthly_income NUMERIC(14,2) NOT NULL DEFAULT 0,
  payday_day_of_month INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE financial_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  monthly_expenses NUMERIC(14,2) NOT NULL DEFAULT 0,
  existing_debt_repayments NUMERIC(14,2) NOT NULL DEFAULT 0,
  available_deposit NUMERIC(14,2) NOT NULL DEFAULT 0,
  desired_property_price NUMERIC(14,2),
  desired_loan_amount NUMERIC(14,2),
  preferred_tenor_years INT,
  preferred_repayment_frequency TEXT DEFAULT 'monthly',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL UNIQUE,
  purpose TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lenders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  website TEXT,
  contact_info JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  verification_status verification_status NOT NULL DEFAULT 'needs_verification',
  last_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE mortgage_products (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lender_id UUID NOT NULL REFERENCES lenders(id),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  mortgage_type TEXT NOT NULL,
  min_loan_amount NUMERIC(14,2),
  max_loan_amount NUMERIC(14,2),
  min_property_value NUMERIC(14,2),
  max_property_value NUMERIC(14,2),
  min_income NUMERIC(14,2),
  max_age INT,
  max_tenor_years INT,
  min_equity_pct NUMERIC(5,2),
  interest_rate NUMERIC(6,3),
  interest_rate_type TEXT NOT NULL DEFAULT 'fixed',
  repayment_frequency TEXT NOT NULL DEFAULT 'monthly',
  processing_fee NUMERIC(14,2),
  valuation_fee NUMERIC(14,2),
  legal_fee NUMERIC(14,2),
  insurance_requirements TEXT,
  acceptable_employment_types JSONB NOT NULL DEFAULT '["salaried","civil_servant"]',
  acceptable_income_types JSONB NOT NULL DEFAULT '["salary"]',
  geographic_restrictions JSONB NOT NULL DEFAULT '[]',
  property_requirements TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  source TEXT,
  source_url TEXT,
  verification_status verification_status NOT NULL DEFAULT 'needs_verification',
  last_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE mortgage_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id UUID NOT NULL REFERENCES mortgage_products(id) ON DELETE CASCADE,
  field TEXT NOT NULL,
  operator TEXT NOT NULL,
  value_type TEXT NOT NULL DEFAULT 'number',
  value JSONB NOT NULL,
  severity TEXT NOT NULL DEFAULT 'hard',
  message_template TEXT NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_types (
  code TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  category TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mortgage_product_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id UUID NOT NULL REFERENCES mortgage_products(id) ON DELETE CASCADE,
  document_type_code TEXT NOT NULL REFERENCES document_types(code),
  label TEXT NOT NULL,
  category TEXT NOT NULL,
  required BOOLEAN NOT NULL DEFAULT TRUE,
  instructions TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE platform_settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by UUID REFERENCES users(id)
);

CREATE TABLE legal_disclaimers (
  key TEXT PRIMARY KEY,
  body TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT 'en-NG',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by UUID REFERENCES users(id)
);

CREATE TABLE eligibility_assessments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  input_snapshot JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'in_progress',
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE eligibility_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assessment_id UUID NOT NULL REFERENCES eligibility_assessments(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES mortgage_products(id),
  outcome eligibility_outcome NOT NULL,
  detail JSONB NOT NULL DEFAULT '{}',
  explanation TEXT NOT NULL DEFAULT '',
  estimated_monthly_repayment NUMERIC(14,2),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE readiness_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assessment_id UUID NOT NULL REFERENCES eligibility_assessments(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  total_score INT NOT NULL,
  components JSONB NOT NULL,
  narrative TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE salary_account_analyses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  assessment_id UUID REFERENCES eligibility_assessments(id),
  document_id UUID,
  declared_salary NUMERIC(14,2),
  median_credit NUMERIC(14,2),
  months_found INT NOT NULL DEFAULT 0,
  gaps JSONB NOT NULL DEFAULT '[]',
  credits JSONB NOT NULL DEFAULT '[]',
  variance_pct NUMERIC(6,2),
  confidence NUMERIC(4,3),
  model_name TEXT,
  model_version TEXT,
  raw_extraction JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mortgage_applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  assessment_id UUID REFERENCES eligibility_assessments(id),
  preferred_product_id UUID REFERENCES mortgage_products(id),
  status application_status NOT NULL DEFAULT 'NEW',
  assigned_advisor_id UUID REFERENCES users(id),
  next_action_text TEXT NOT NULL DEFAULT '',
  automation_level automation_level NOT NULL DEFAULT 'suggest_only',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE application_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  actor_id UUID REFERENCES users(id),
  event_type TEXT NOT NULL,
  from_status application_status,
  to_status application_status,
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE advisor_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  advisor_id UUID NOT NULL REFERENCES users(id),
  assigned_by UUID REFERENCES users(id),
  active BOOLEAN NOT NULL DEFAULT TRUE,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE advisor_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES users(id),
  body TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'internal',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE concierge_suggestions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_id UUID NOT NULL REFERENCES mortgage_applications(id) ON DELETE CASCADE,
  suggestion_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  confidence NUMERIC(4,3),
  status TEXT NOT NULL DEFAULT 'pending',
  reviewed_by UUID REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  application_id UUID REFERENCES mortgage_applications(id),
  document_type_code TEXT NOT NULL REFERENCES document_types(code),
  storage_key TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  checksum TEXT,
  version INT NOT NULL DEFAULT 1,
  status document_status NOT NULL DEFAULT 'uploaded',
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  reviewer_id UUID REFERENCES users(id),
  decision TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  channel TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}',
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE content_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  type TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id UUID REFERENCES users(id),
  action TEXT NOT NULL,
  entity_type TEXT,
  entity_id UUID,
  ip TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE analytics_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_name TEXT NOT NULL,
  user_id UUID REFERENCES users(id),
  session_id TEXT,
  properties JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mortgage_products_lender ON mortgage_products(lender_id);
CREATE INDEX idx_mortgage_rules_product ON mortgage_rules(product_id);
CREATE INDEX idx_assessments_user ON eligibility_assessments(user_id);
CREATE INDEX idx_applications_user ON mortgage_applications(user_id);
CREATE INDEX idx_applications_status ON mortgage_applications(status);
CREATE INDEX idx_documents_user ON documents(user_id);
CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at);
