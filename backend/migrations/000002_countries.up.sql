CREATE TABLE countries (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  currency_code TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT 'en',
  region_label TEXT NOT NULL DEFAULT 'Region',
  regions JSONB NOT NULL DEFAULT '[]',
  default_iti_pct NUMERIC(5,2) NOT NULL DEFAULT 35,
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'coming_soon', 'inactive')),
  sort_order INT NOT NULL DEFAULT 100,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE lenders
  ADD COLUMN country_code TEXT REFERENCES countries(code);

ALTER TABLE mortgage_products
  ADD COLUMN country_code TEXT REFERENCES countries(code);

ALTER TABLE user_profiles
  ADD COLUMN country_code TEXT REFERENCES countries(code);

ALTER TABLE eligibility_assessments
  ADD COLUMN country_code TEXT REFERENCES countries(code);

INSERT INTO countries (code, name, currency_code, locale, region_label, regions, default_iti_pct, status, sort_order)
VALUES (
  'NG',
  'Nigeria',
  'NGN',
  'en-NG',
  'State',
  '["Abia","Adamawa","Akwa Ibom","Anambra","Bauchi","Bayelsa","Benue","Borno","Cross River","Delta","Ebonyi","Edo","Ekiti","Enugu","FCT","Gombe","Imo","Jigawa","Kaduna","Kano","Katsina","Kebbi","Kogi","Kwara","Lagos","Nasarawa","Niger","Ogun","Ondo","Osun","Oyo","Plateau","Rivers","Sokoto","Taraba","Yobe","Zamfara"]'::jsonb,
  35,
  'active',
  10
);

-- Placeholder markets: activate + seed products when ready to launch.
INSERT INTO countries (code, name, currency_code, locale, region_label, regions, default_iti_pct, status, sort_order)
VALUES
  ('GH', 'Ghana', 'GHS', 'en-GH', 'Region', '[]'::jsonb, 35, 'coming_soon', 20),
  ('KE', 'Kenya', 'KES', 'en-KE', 'County', '[]'::jsonb, 35, 'coming_soon', 30);

UPDATE lenders SET country_code = 'NG' WHERE country_code IS NULL;
UPDATE mortgage_products SET country_code = 'NG' WHERE country_code IS NULL;
UPDATE user_profiles SET country_code = 'NG' WHERE country_code IS NULL;
UPDATE eligibility_assessments SET country_code = 'NG' WHERE country_code IS NULL;

ALTER TABLE lenders ALTER COLUMN country_code SET NOT NULL;
ALTER TABLE mortgage_products ALTER COLUMN country_code SET NOT NULL;

CREATE INDEX lenders_country_idx ON lenders (country_code) WHERE deleted_at IS NULL;
CREATE INDEX mortgage_products_country_idx ON mortgage_products (country_code) WHERE deleted_at IS NULL;

INSERT INTO platform_settings (key, value)
VALUES ('default_country_code', '"NG"')
ON CONFLICT (key) DO NOTHING;
