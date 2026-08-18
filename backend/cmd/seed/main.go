package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/homegauge/homegauge/backend/internal/config"
	"github.com/homegauge/homegauge/backend/internal/platform/db"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if err := seedDocumentTypes(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
	if err := seedSettings(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
	if err := seedDisclaimers(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
	if err := seedUsers(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
	if err := seedLendersAndProducts(ctx, sqlDB); err != nil {
		log.Fatal(err)
	}
	fmt.Println("HomeGauge seed complete")
}

func seedDocumentTypes(ctx context.Context, db *sql.DB) error {
	rows := [][3]string{
		{"valid_id", "Valid government ID", "identity"},
		{"passport_photo", "Passport photograph", "identity"},
		{"payslips_3m", "3 months’ payslips", "income"},
		{"employment_letter", "Employment / introduction letter", "income"},
		{"salary_statements_6m", "6 months’ salary account statements", "banking"},
		{"nhf_evidence", "NHF contribution evidence", "income"},
		{"offer_letter", "Property offer letter", "property"},
		{"title_docs", "Registered title documents", "property"},
	}
	for _, r := range rows {
		_, err := db.ExecContext(ctx, `
			INSERT INTO document_types (code, label, category)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO NOTHING
		`, r[0], r[1], r[2])
		if err != nil {
			return err
		}
	}
	return nil
}

func seedSettings(ctx context.Context, db *sql.DB) error {
	settings := map[string]string{
		"salary_variance_pct":      `15`,
		"salary_payday_last_days":  `7`,
		"default_iti_pct":          `35`,
		"default_country_code":     `"NG"`,
		"automation_level":         `"suggest_only"`,
		"readiness_weights": `{
			"salary_pattern": 30,
			"deposit_readiness": 20,
			"debt_iti": 20,
			"documentation": 15,
			"employment_tenure": 10,
			"eligibility_coverage": 5
		}`,
	}
	for k, v := range settings {
		_, err := db.ExecContext(ctx, `
			INSERT INTO platform_settings (key, value, updated_at)
			VALUES ($1, $2::jsonb, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func seedDisclaimers(ctx context.Context, db *sql.DB) error {
	body := `HomeGauge is not a bank or lender. Eligibility results are educational estimates based on information you provide and publicly stated product criteria. Only a licensed lender can approve a mortgage.`
	_, err := db.ExecContext(ctx, `
		INSERT INTO legal_disclaimers (key, body, updated_at)
		VALUES ('eligibility_general', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET body = EXCLUDED.body, updated_at = NOW()
	`, body)
	return err
}

func seedUsers(ctx context.Context, db *sql.DB) error {
	type u struct {
		email, password, role, name string
	}
	users := []u{
		{"admin@homegauge.local", "ChangeMeAdmin1!", "ADMIN", "HomeGauge Admin"},
		{"advisor@homegauge.local", "ChangeMeAdvisor1!", "ADVISOR", "HomeGauge Advisor"},
		{"demo@homegauge.local", "ChangeMeDemo1!", "CUSTOMER", "Demo Customer"},
		{"lender@homegauge.local", "ChangeMeLender1!", "LENDER_USER", "HomeGauge Lender"},
	}
	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		var existing string
		err = db.QueryRowContext(ctx, `SELECT id::text FROM users WHERE LOWER(email)=LOWER($1) AND deleted_at IS NULL`, u.email).Scan(&existing)
		if err == nil {
			_, err = db.ExecContext(ctx, `
				UPDATE users
				SET password_hash=$2, role=$3, status='active', email_verified_at=COALESCE(email_verified_at, NOW()), updated_at=NOW()
				WHERE id=$1::uuid
			`, existing, string(hash), u.role)
			if err != nil {
				return err
			}
			_, _ = db.ExecContext(ctx, `UPDATE user_profiles SET full_name=$2, updated_at=NOW() WHERE user_id=$1::uuid`, existing, u.name)
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		var id string
		if err := db.QueryRowContext(ctx, `
			INSERT INTO users (email, password_hash, role, email_verified_at)
			VALUES ($1, $2, $3, NOW())
			RETURNING id::text
		`, u.email, string(hash), u.role).Scan(&id); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO user_profiles (user_id, full_name) VALUES ($1::uuid, $2)`, id, u.name); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO employment_profiles (user_id) VALUES ($1::uuid)`, id); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO financial_profiles (user_id) VALUES ($1::uuid)`, id); err != nil {
			return err
		}
	}
	return nil
}

func seedLendersAndProducts(ctx context.Context, db *sql.DB) error {
	now := time.Now()
	const country = "NG"

	var fmbnID, stanbicID, commercialID string
	if err := upsertLender(ctx, db, &fmbnID, country, "Federal Mortgage Bank of Nigeria (FMBN) / NHF channel",
		"Manages the National Housing Fund (NHF) social mortgage channel, typically accessed via Primary Mortgage Banks.",
		"https://www.fmbn.gov.ng", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &stanbicID, country, "Stanbic IBTC Bank",
		"Commercial bank offering MREIF and other home loan variants. Terms below taken from lender-published MREIF FAQ.",
		"https://www.stanbicibtcbank.com", "verified", &now); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &commercialID, country, "Commercial mortgage (market indicative)",
		"Placeholder for typical commercial bank mortgages. Rates change with policy rates — mark verified only after confirming a specific bank offer.",
		"", "needs_verification", nil); err != nil {
		return err
	}

	var nhfProduct string
	if err := upsertProduct(ctx, db, &nhfProduct, country, fmbnID, "NHF Mortgage Loan",
		"Social mortgage for NHF contributors, usually via a Primary Mortgage Bank. Interest commonly cited at 6% p.a.; max loan up to 50m NGN; long tenor up to 30 years subject to age. Equity often 0–10% by loan band — confirm current FMBN circular.",
		"nhf", 1_000_000, 50_000_000, 6.0, 6.0, 6.0, "fixed", 30, 10, 18, 60,
		"FMBN / PMB public materials", "https://www.fmbn.gov.ng", "needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, nhfProduct, []ruleSeed{
		{"monthly_income", "gte", 300000, "hard", "This product usually needs enough income to keep repayments affordable (often around one-third of income)."},
		{"age", "lte", 60, "hard", "You should typically complete repayment before age 60 / retirement."},
		{"employment_type", "in", []string{"salaried", "civil_servant"}, "hard", "HomeGauge’s automated review currently supports salary-account workers."},
		{"salary_months", "gte", 6, "hard", "We look for about 6 months of clear salary credits on one account."},
		{"equity_pct", "gte", 10, "soft", "Many NHF loan bands ask for around {value}% equity (lower bands may ask less — confirm with the PMB)."},
		{"nhf_contributor_months", "gte", 6, "hard", "NHF loans usually require at least 6 months of NHF contributions."},
	})
	_ = replaceDocs(ctx, db, nhfProduct, []docSeed{
		{"valid_id", "Valid ID", "identity", true},
		{"payslips_3m", "3 months’ payslips", "income", true},
		{"employment_letter", "Employment letter", "income", true},
		{"salary_statements_6m", "6 months’ salary account statements", "banking", true},
		{"nhf_evidence", "NHF contribution evidence", "income", true},
		{"offer_letter", "Property offer letter", "property", true},
		{"title_docs", "Title documents", "property", true},
	})

	var mreif string
	if err := upsertProduct(ctx, db, &mreif, country, stanbicID, "MREIF Home Loan",
		"Ministry of Finance Incorporated Real Estate Investment Fund home loan via Stanbic IBTC. Lender FAQ: 9.75% p.a., 10% equity, 10m–100m NGN, up to 20 years, ITI 35%, salary domiciliation required for salaried applicants.",
		"mreif", 10_000_000, 100_000_000, 9.75, 9.75, 9.75, "fixed", 20, 10, 500000, 60,
		"Stanbic IBTC MREIF FAQ", "https://www.stanbicibtcbank.com/nigeriabank/personal/products-and-services/all-loans/MREIF-Frequently-Asked-Questions",
		"verified", &now); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, mreif, []ruleSeed{
		{"monthly_income", "gte", 500000, "hard", "Specialised/MREIF variants often cite a minimum net income around {value}."},
		{"age", "lte", 60, "hard", "Applicants are typically 21–60 (or retirement) at loan maturity."},
		{"employment_type", "in", []string{"salaried"}, "hard", "Automated review is for salaried applicants with salary credits."},
		{"years_employed", "gte", 0.5, "hard", "Usually at least 6 months with your current employer."},
		{"salary_months", "gte", 6, "hard", "Provide 6 months’ bank statements showing salary."},
		{"equity_pct", "gte", 10, "hard", "Minimum equity contribution is typically {value}%."},
		{"loan_amount", "gte", 10000000, "hard", "Minimum loan amount is typically {value}."},
		{"loan_amount", "lte", 100000000, "hard", "Maximum standard loan amount is typically {value}."},
		{"iti_pct", "lte", 35, "hard", "Installment-to-income (ITI) should usually stay at or below {value}%."},
	})
	_ = replaceDocs(ctx, db, mreif, []docSeed{
		{"valid_id", "Valid photo ID", "identity", true},
		{"payslips_3m", "3 months’ payslips", "income", true},
		{"employment_letter", "Employer introduction letter", "income", true},
		{"salary_statements_6m", "6 months’ statements with salary evidence", "banking", true},
		{"offer_letter", "Offer letter from vendor", "property", true},
		{"title_docs", "Registered title & survey", "property", true},
	})

	var comm string
	if err := upsertProduct(ctx, db, &comm, country, commercialID, "Commercial bank mortgage (indicative)",
		"Typical commercial mortgages in a high-rate environment often price roughly from 20–26% p.a. (indicative 24%), with 20–30% equity and 10–20 year tenors. The actual rate is negotiated with the lender — this is not an offer.",
		"commercial", 5_000_000, 150_000_000, 24.0, 20.0, 26.0, "negotiable", 15, 25, 750000, 55,
		"Market indicative 2026", "", "needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, comm, []ruleSeed{
		{"monthly_income", "gte", 750000, "hard", "Commercial mortgages usually need strong verifiable salary income."},
		{"employment_type", "in", []string{"salaried", "civil_servant"}, "hard", "Salary-account review only in this MVP."},
		{"salary_months", "gte", 6, "hard", "6 months of salary credits required for automated review."},
		{"equity_pct", "gte", 25, "hard", "Equity of about {value}% is common for commercial mortgages."},
		{"iti_pct", "lte", 35, "soft", "Keeping repayments near or below {value}% of income improves fit."},
		{"age", "lte", 55, "soft", "Many banks prefer repayment to finish by mid-50s / retirement."},
	})
	_ = replaceDocs(ctx, db, comm, []docSeed{
		{"valid_id", "Valid ID", "identity", true},
		{"payslips_3m", "3 months’ payslips", "income", true},
		{"employment_letter", "Employment letter", "income", true},
		{"salary_statements_6m", "6 months’ salary statements", "banking", true},
		{"offer_letter", "Offer letter", "property", true},
		{"title_docs", "Title documents", "property", true},
	})

	_, err := db.ExecContext(ctx, `
		UPDATE users SET lender_id = $1::uuid, updated_at = NOW()
		WHERE LOWER(email) = LOWER('lender@homegauge.local') AND deleted_at IS NULL
	`, stanbicID)
	return err
}

func upsertLender(ctx context.Context, db *sql.DB, id *string, country, name, desc, website, vstatus string, verifiedAt *time.Time) error {
	var verified sql.NullTime
	if verifiedAt != nil {
		verified = sql.NullTime{Time: *verifiedAt, Valid: true}
	}
	err := db.QueryRowContext(ctx, `SELECT id::text FROM lenders WHERE name = $1 AND country_code = $2 AND deleted_at IS NULL`, name, country).Scan(id)
	if err == nil {
		_, err = db.ExecContext(ctx, `
			UPDATE lenders SET description=$2, website=NULLIF($3,''), verification_status=$4::verification_status, last_verified_at=$5, country_code=$6, updated_at=NOW()
			WHERE id=$1::uuid`, *id, desc, website, vstatus, verified, country)
		return err
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO lenders (name, description, website, verification_status, last_verified_at, country_code)
		VALUES ($1,$2,NULLIF($3,''),$4::verification_status,$5,$6) RETURNING id::text
	`, name, desc, website, vstatus, verified, country).Scan(id)
}

func upsertProduct(ctx context.Context, db *sql.DB, id *string, country, lenderID, name, desc, mtype string,
	minLoan, maxLoan, rate, rateMin, rateMax float64, rateType string, tenor int, equity, minIncome float64, maxAge int,
	source, sourceURL, vstatus string, verifiedAt *time.Time) error {
	var verified sql.NullTime
	if verifiedAt != nil {
		verified = sql.NullTime{Time: *verifiedAt, Valid: true}
	}
	err := db.QueryRowContext(ctx, `SELECT id::text FROM mortgage_products WHERE name=$1 AND lender_id=$2::uuid AND deleted_at IS NULL`, name, lenderID).Scan(id)
	if err == nil {
		_, err = db.ExecContext(ctx, `
			UPDATE mortgage_products SET description=$2, mortgage_type=$3, min_loan_amount=$4, max_loan_amount=$5,
			interest_rate=$6, interest_rate_min=$7, interest_rate_max=$8, interest_rate_type=$9,
			max_tenor_years=$10, min_equity_pct=$11, min_income=$12, max_age=$13,
			source=NULLIF($14,''), source_url=NULLIF($15,''), verification_status=$16::verification_status, last_verified_at=$17, country_code=$18, updated_at=NOW()
			WHERE id=$1::uuid`, *id, desc, mtype, minLoan, maxLoan, rate, rateMin, rateMax, rateType, tenor, equity, minIncome, maxAge, source, sourceURL, vstatus, verified, country)
		return err
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO mortgage_products (
			lender_id, name, description, mortgage_type, min_loan_amount, max_loan_amount,
			interest_rate, interest_rate_min, interest_rate_max, interest_rate_type, max_tenor_years, min_equity_pct, min_income, max_age,
			source, source_url, verification_status, last_verified_at, country_code
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,''),NULLIF($16,''),$17::verification_status,$18,$19)
		RETURNING id::text
	`, lenderID, name, desc, mtype, minLoan, maxLoan, rate, rateMin, rateMax, rateType, tenor, equity, minIncome, maxAge, source, sourceURL, vstatus, verified, country).Scan(id)
}

type ruleSeed struct {
	field, op string
	value     any
	severity, msg string
}

func replaceRules(ctx context.Context, db *sql.DB, productID string, rules []ruleSeed) error {
	_, _ = db.ExecContext(ctx, `DELETE FROM mortgage_rules WHERE product_id=$1::uuid`, productID)
	for i, r := range rules {
		val := fmt.Sprintf("%v", r.value)
		switch v := r.value.(type) {
		case []string:
			b := "["
			for i, s := range v {
				if i > 0 {
					b += ","
				}
				b += fmt.Sprintf("%q", s)
			}
			b += "]"
			val = b
		case float64, int:
			val = fmt.Sprintf("%v", v)
		case string:
			val = fmt.Sprintf("%q", v)
		}
		_, err := db.ExecContext(ctx, `
			INSERT INTO mortgage_rules (product_id, field, operator, value_type, value, severity, message_template, sort_order)
			VALUES ($1::uuid,$2,$3,'auto',$4::jsonb,$5,$6,$7)
		`, productID, r.field, r.op, val, r.severity, r.msg, i)
		if err != nil {
			return err
		}
	}
	return nil
}

type docSeed struct {
	code, label, category string
	required              bool
}

func replaceDocs(ctx context.Context, db *sql.DB, productID string, docs []docSeed) error {
	_, _ = db.ExecContext(ctx, `DELETE FROM mortgage_product_documents WHERE product_id=$1::uuid`, productID)
	for _, d := range docs {
		_, err := db.ExecContext(ctx, `
			INSERT INTO mortgage_product_documents (product_id, document_type_code, label, category, required)
			VALUES ($1::uuid,$2,$3,$4,$5)
		`, productID, d.code, d.label, d.category, d.required)
		if err != nil {
			return err
		}
	}
	return nil
}
