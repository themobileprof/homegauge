UPDATE mortgage_products SET country_code = NULL;
UPDATE lenders SET country_code = NULL;
ALTER TABLE eligibility_assessments DROP COLUMN IF EXISTS country_code;
ALTER TABLE user_profiles DROP COLUMN IF EXISTS country_code;
ALTER TABLE mortgage_products DROP COLUMN IF EXISTS country_code;
ALTER TABLE lenders DROP COLUMN IF EXISTS country_code;
DELETE FROM platform_settings WHERE key = 'default_country_code';
DROP TABLE IF EXISTS countries;
