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
		{"office_id", "Office identity card", "identity"},
		{"passport_photo", "Passport photograph", "identity"},
		{"utility_bill", "Recent utility bill / tenancy proof", "identity"},
		{"payslips_3m", "3 months' payslips", "income"},
		{"employment_letter", "Employment / introduction letter", "income"},
		{"employer_intro_letter", "Letter of introduction from employer", "income"},
		{"salary_statements_6m", "6 months' salary account statements", "banking"},
		{"account_statements_12m", "12 months' account statements", "banking"},
		{"nhf_evidence", "NHF contribution evidence", "income"},
		{"pension_statement", "Pension statement of account", "income"},
		{"tax_clearance", "Current tax clearance certificate", "income"},
		{"offer_letter", "Property offer letter", "property"},
		{"title_docs", "Registered title documents", "property"},
		{"building_plan", "Approved building plan", "property"},
		{"valuation_report", "Property valuation report", "property"},
		{"gsi_form", "Duly executed Global Standing Instruction (GSI) form", "banking"},
		{"direct_debit_mandate", "Direct debit mandate and post-dated cheque schedule", "banking"},
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
		"salary_variance_pct":     `15`,
		"salary_payday_last_days": `7`,
		"default_iti_pct":         `35`,
		"default_country_code":    `"NG"`,
		"automation_level":        `"suggest_only"`,
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

	// ── Lenders ──────────────────────────────────────────────────────────────
	var fmbnID, stanbicID, accessID, firstBankID, gtbID, zenithID, abbeyID, lbicID, mixtaID, infinityID string

	if err := upsertLender(ctx, db, &fmbnID, country,
		"Federal Mortgage Bank of Nigeria (FMBN) / NHF channel",
		"Manages the National Housing Fund (NHF) social mortgage channel, typically accessed via Primary Mortgage Banks.",
		"https://www.fmbn.gov.ng", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &stanbicID, country,
		"Stanbic IBTC Bank",
		"Commercial bank offering MREIF and other home loan variants. Terms from lender-published MREIF FAQ (2024).",
		"https://www.stanbicibtcbank.com", "verified", &now); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &accessID, country,
		"Access Bank",
		"Offers residential mortgage loans and the EasyHome product range for salaried customers. Rates typically negotiated with the branch.",
		"https://www.accessbankplc.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &firstBankID, country,
		"First Bank of Nigeria",
		"Provides mortgage finance under the FirstHome and other housing-loan products for confirmed employees with salary domiciliation.",
		"https://www.firstbanknigeria.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &gtbID, country,
		"Guaranty Trust Bank (GTBank)",
		"Offers home loans for outright purchase and construction. Eligibility linked to salary-account domiciliation and employer confirmation letter.",
		"https://www.gtbank.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &zenithID, country,
		"Zenith Bank",
		"Provides residential mortgage finance for purchase and construction, targeted at salaried customers in the public and private sector.",
		"https://www.zenithbank.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &abbeyID, country,
		"Abbey Mortgage Bank",
		"Primary Mortgage Institution (PMI) licensed by the CBN. Offers NHF, FMBN, and retail mortgage products. One of the oldest dedicated mortgage banks in Nigeria.",
		"https://www.abbeymortgagebank.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &lbicID, country,
		"Lagos Building Investment Company (LBIC)",
		"Lagos state-backed mortgage institution. Publishes a mortgage checklist and customer onboarding forms for walk-in and formal mortgage processing.",
		"https://lbicplc.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &mixtaID, country,
		"Mixta Africa (DUO)",
		"Developer-led rent-to-own housing model where annual rent contributes toward eventual ownership conversion.",
		"https://duo.mixtafrica.com", "needs_verification", nil); err != nil {
		return err
	}
	if err := upsertLender(ctx, db, &infinityID, country,
		"Infinity Trust Mortgage Bank",
		"Primary mortgage bank offering mortgage loans with institution-specific documentation, underwriting, and perfection workflow.",
		"https://itmbplc.com", "needs_verification", nil); err != nil {
		return err
	}

	// ── Shared building blocks ────────────────────────────────────────────────
	standardDocs := []docSeed{
		{"valid_id", "Valid government-issued photo ID", "identity", true},
		{"payslips_3m", "3 months' payslips", "income", true},
		{"employment_letter", "Employment / confirmation letter", "income", true},
		{"salary_statements_6m", "6 months' salary account statements", "banking", true},
		{"offer_letter", "Property offer letter", "property", true},
		{"title_docs", "Registered title documents", "property", true},
	}
	// Optional supporting assets for employed applicants to strengthen files.
	salarySupportDocs := []docSeed{
		{"pension_statement", "Pension statement of account", "income", false},
		{"tax_clearance", "Current tax clearance certificate", "income", false},
		{"account_statements_12m", "Additional 12 months account statements", "banking", false},
	}
	standardRules := []ruleSeed{
		{"employment_type", "in", []string{"salaried", "civil_servant"}, "hard", "Automated review is for salaried workers with a clear salary credit."},
		{"salary_months", "gte", 6, "hard", "We look for about 6 months of salary credits on one account."},
		{"iti_pct", "lte", 35, "soft", "Keeping repayments at or below {value}% of income improves fit."},
		{"age", "lte", 60, "hard", "You should typically complete repayment before age 60."},
	}

	// ── 1. NHF Mortgage Loan (FMBN) ──────────────────────────────────────────
	var nhfProduct string
	if err := upsertProduct(ctx, db, &nhfProduct, country, fmbnID, "NHF Mortgage Loan",
		"Social mortgage for NHF contributors, via a Primary Mortgage Bank (PMB). Interest commonly cited at 6% p.a.; max loan up to ₦50m; tenor up to 30 years subject to age. Equity requirement varies by loan band — confirm the current FMBN circular with your PMB.",
		"nhf", 1_000_000, 50_000_000, 6.0, 6.0, 6.0, "fixed", 30, 10, 18_000, 60,
		"active", "FMBN / PMB public materials", "https://www.fmbn.gov.ng", "needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, nhfProduct, append([]ruleSeed{
		{"monthly_income", "gte", 18000, "hard", "NHF scheme has a low income floor; primary eligibility is NHF contribution history."},
		{"equity_pct", "gte", 10, "soft", "Many NHF loan bands ask for around {value}% equity (lower bands may ask less)."},
		{"nhf_contributor_months", "gte", 6, "hard", "NHF loans require at least 6 months of NHF contributions."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, nhfProduct, append(append([]docSeed{}, standardDocs...), append(salarySupportDocs, docSeed{"nhf_evidence", "NHF contribution evidence (EID card or FMBN statement)", "income", true})...))

	// ── 2. Stanbic IBTC — MREIF Home Loan ────────────────────────────────────
	var mreif string
	if err := upsertProduct(ctx, db, &mreif, country, stanbicID, "MREIF Home Loan",
		"Ministry of Finance Incorporated Real Estate Investment Fund home loan via Stanbic IBTC. Published terms: 9.75% p.a., 10% equity, ₦10m–₦100m, up to 20 years, ITI ≤35%, salary domiciliation required.",
		"mreif", 10_000_000, 100_000_000, 9.75, 9.75, 9.75, "fixed", 20, 10, 500_000, 60,
		"active", "Stanbic IBTC MREIF FAQ",
		"https://www.stanbicibtcbank.com/nigeriabank/personal/products-and-services/all-loans/MREIF-Frequently-Asked-Questions",
		"verified", &now); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, mreif, []ruleSeed{
		{"monthly_income", "gte", 500000, "hard", "MREIF variants typically cite a minimum net income around {value}."},
		{"age", "lte", 60, "hard", "Applicants are typically 21–60 at loan maturity."},
		{"employment_type", "in", []string{"salaried"}, "hard", "Automated review is for salaried applicants with salary credits."},
		{"years_employed", "gte", 0.5, "hard", "Usually at least 6 months with your current employer."},
		{"salary_months", "gte", 6, "hard", "Provide 6 months' bank statements showing salary."},
		{"equity_pct", "gte", 10, "hard", "Minimum equity contribution is typically {value}%."},
		{"loan_amount", "gte", 10000000, "hard", "Minimum loan amount is typically {value}."},
		{"loan_amount", "lte", 100000000, "hard", "Maximum standard loan amount is {value}."},
		{"iti_pct", "lte", 35, "hard", "ITI should stay at or below {value}%."},
	})
	_ = replaceDocs(ctx, db, mreif, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 3. Access Bank — EasyHome Mortgage ───────────────────────────────────
	var accessProduct string
	if err := upsertProduct(ctx, db, &accessProduct, country, accessID, "Access Bank EasyHome Mortgage",
		"Residential purchase or construction mortgage for salaried Access Bank customers. Rates are typically negotiated; indicative range 20–24% p.a. in the current environment. Salary domiciliation with Access Bank is usually required.",
		"commercial", 5_000_000, 100_000_000, 22.0, 20.0, 24.0, "negotiable", 20, 20, 400_000, 60,
		"active", "Access Bank product page", "https://www.accessbankplc.com/personal/loans/mortgages",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, accessProduct, append([]ruleSeed{
		{"monthly_income", "gte", 400000, "hard", "Access Bank typically requires a minimum verifiable net income around {value}."},
		{"equity_pct", "gte", 20, "hard", "Equity of about {value}% is typically required."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, accessProduct, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 4. First Bank — FirstHome Mortgage ───────────────────────────────────
	var firstBankProduct string
	if err := upsertProduct(ctx, db, &firstBankProduct, country, firstBankID, "First Bank FirstHome Loan",
		"Residential mortgage for confirmed employees with salary domiciliation at First Bank. Rates are negotiable and vary with the CBN MPR; indicative range 21–25% p.a. Employer confirmation and 6 months' statements required.",
		"commercial", 5_000_000, 150_000_000, 23.0, 21.0, 25.0, "negotiable", 20, 25, 500_000, 60,
		"active", "First Bank product page", "https://www.firstbanknigeria.com/personal/loans/mortgage",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, firstBankProduct, append([]ruleSeed{
		{"monthly_income", "gte", 500000, "hard", "First Bank mortgage products typically require a minimum net income of around {value}."},
		{"equity_pct", "gte", 25, "hard", "Equity contribution of about {value}% is common."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, firstBankProduct, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 5. GTBank — Home Loan ─────────────────────────────────────────────────
	var gtbProduct string
	if err := upsertProduct(ctx, db, &gtbProduct, country, gtbID, "GTBank Home Loan",
		"Purchase or construction mortgage for salaried GTBank customers. Eligibility tied to salary-account domiciliation and employer letter. Rates are negotiated; indicative range 22–26% p.a.",
		"commercial", 5_000_000, 150_000_000, 24.0, 22.0, 26.0, "negotiable", 15, 25, 600_000, 60,
		"active", "GTBank product page", "https://www.gtbank.com/personal/loans/mortgage",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, gtbProduct, append([]ruleSeed{
		{"monthly_income", "gte", 600000, "hard", "GTBank mortgage products typically require a minimum net income of around {value}."},
		{"equity_pct", "gte", 25, "hard", "Equity contribution of about {value}% is typical."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, gtbProduct, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 6. Zenith Bank — Residential Mortgage ────────────────────────────────
	var zenithProduct string
	if err := upsertProduct(ctx, db, &zenithProduct, country, zenithID, "Zenith Bank Residential Mortgage",
		"Home-purchase and construction finance for confirmed salaried employees. Rates are negotiable; indicative 21–25% p.a. Salary domiciliation at Zenith Bank is typically required.",
		"commercial", 5_000_000, 150_000_000, 23.0, 21.0, 25.0, "negotiable", 20, 25, 500_000, 60,
		"active", "Zenith Bank product page", "https://www.zenithbank.com/personal/mortgage",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, zenithProduct, append([]ruleSeed{
		{"monthly_income", "gte", 500000, "hard", "Zenith Bank mortgage products typically require a minimum net income around {value}."},
		{"equity_pct", "gte", 25, "hard", "Equity contribution of about {value}% is commonly required."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, zenithProduct, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 7. Abbey Mortgage Bank — Retail Home Loan ────────────────────────────
	var abbeyProduct string
	if err := upsertProduct(ctx, db, &abbeyProduct, country, abbeyID, "Abbey Mortgage Bank Home Loan",
		"Retail mortgage from one of Nigeria's leading Primary Mortgage Institutions. Offers NHF-route and non-NHF residential finance. Rates on commercial products are indicative 18–22% p.a.; NHF route follows FMBN circular.",
		"commercial", 2_000_000, 50_000_000, 20.0, 18.0, 22.0, "negotiable", 20, 20, 300_000, 60,
		"active", "Abbey Mortgage Bank product page", "https://www.abbeymortgagebank.com",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, abbeyProduct, append([]ruleSeed{
		{"monthly_income", "gte", 300000, "hard", "Abbey Mortgage typically requires a minimum verifiable income of about {value}."},
		{"equity_pct", "gte", 20, "hard", "Equity contribution of about {value}% is typical for retail products."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, abbeyProduct, append(append([]docSeed{}, standardDocs...), salarySupportDocs...))

	// ── 8. LBIC — Home Ownership Mortgage ────────────────────────────────────
	var lbicProduct string
	if err := upsertProduct(ctx, db, &lbicProduct, country, lbicID, "LBIC Home Ownership Mortgage",
		"Lagos Building Investment Company mortgage lane for salaried and structured-income applicants. Onboarding references LBIC mortgage checklist and KYC forms; exact pricing and tenor are branch-confirmed.",
		"commercial", 3_000_000, 120_000_000, 22.0, 19.0, 25.0, "negotiable", 20, 20, 250_000, 60,
		"active", "LBIC mortgage checklist", "https://lbicplc.com/mortage-checklist",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, lbicProduct, append([]ruleSeed{
		{"monthly_income", "gte", 250000, "hard", "LBIC processing expects evidence of stable income around {value} or higher."},
		{"equity_pct", "gte", 20, "hard", "Plan for at least about {value}% equity unless LBIC confirms a lower scheme threshold."},
	}, standardRules...))
	_ = replaceDocs(ctx, db, lbicProduct, append(append([]docSeed{}, standardDocs...), append(salarySupportDocs,
		docSeed{"passport_photo", "Passport photograph", "identity", true},
	)...))

	// ── 9. Mixta DUO — Rent-to-Own Pathway ───────────────────────────────────
	var duoProduct string
	if err := upsertProduct(ctx, db, &duoProduct, country, mixtaID, "Mixta DUO Rent-to-Own (Marula Park)",
		"Developer rent-to-own model in Lagos where initial equity and annual rent can count toward eventual ownership conversion. Not a traditional bank mortgage at entry; conversion can happen via cash-out or mortgage exit.",
		"scheme", 15_000_000, 120_000_000, 0.0, 0.0, 0.0, "negotiable", 3, 5, 300_000, 60,
		"inactive", "Mixta DUO product page", "https://duo.mixtafrica.com/",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, duoProduct, []ruleSeed{
		{"monthly_income", "gte", 300000, "soft", "Higher stable income improves rent-to-own continuity and conversion readiness."},
		{"equity_pct", "gte", 5, "hard", "DUO entry references at least about {value}% initial equity payment."},
		{"employment_type", "in", []string{"salaried", "self_employed", "business_owner"}, "soft", "Structured income history supports this scheme model."},
		{"age", "lte", 60, "soft", "Earlier entry can improve eventual mortgage-conversion options."},
		{"iti_pct", "lte", 40, "soft", "Keep total housing burden near or below {value}% of income for safer conversion."},
	})
	_ = replaceDocs(ctx, db, duoProduct, []docSeed{
		{"valid_id", "Valid government-issued photo ID", "identity", true},
		{"passport_photo", "Passport photograph", "identity", true},
		{"payslips_3m", "Recent payslips or income proof", "income", true},
		{"salary_statements_6m", "6 months' account statements", "banking", true},
	})

	// ── 10. Infinity Trust Mortgage Bank — Mortgage Loan ─────────────────────
	var infinityProduct string
	if err := upsertProduct(ctx, db, &infinityProduct, country, infinityID, "Infinity Trust Mortgage Loan",
		"Mortgage loan checklist supplied by Infinity Trust staff: stated interest rate 31% p.a., max tenor 5 years (subject to age), with pre- and post-approval fees plus strict documentation and mandate requirements.",
		"commercial", 3_000_000, 150_000_000, 31.0, 31.0, 31.0, "fixed", 5, 20, 350_000, 60,
		"active", "Infinity Trust staff mortgage checklist (PDF)", "",
		"needs_verification", nil); err != nil {
		return err
	}
	_ = replaceRules(ctx, db, infinityProduct, []ruleSeed{
		{"monthly_income", "gte", 350000, "hard", "Infinity Trust underwriting expects strong evidenced income; use at least {value} monthly as a screening floor."},
		{"salary_months", "gte", 12, "hard", "Checklist requests 12 months of income/account statements."},
		{"age", "lte", 60, "hard", "Tenor is capped at 5 years and still depends on age at maturity."},
		{"tenor_years", "lte", 5, "hard", "Infinity Trust checklist states maximum tenor of {value} years."},
		{"iti_pct", "lte", 40, "soft", "Keep total repayment burden within {value}% of income for safer approval odds."},
		{"equity_pct", "gte", 20, "soft", "Plan for meaningful equity contribution; confirm current branch threshold."},
	})
	_ = replaceDocs(ctx, db, infinityProduct, []docSeed{
		{"valid_id", "Valid means of ID (NIN/driver's license/passport/voter's card)", "identity", true},
		{"office_id", "Office identity card", "identity", true},
		{"passport_photo", "Two passport photographs", "identity", true},
		{"utility_bill", "Utility bill / tenancy proof matching applicant address", "identity", true},
		{"employment_letter", "Employment offer letter", "income", true},
		{"employer_intro_letter", "Letter of introduction from employer", "income", true},
		{"pension_statement", "Pension statement of account", "income", true},
		{"tax_clearance", "Current tax clearance certificate", "income", true},
		{"account_statements_12m", "12 months income/account statements", "banking", true},
		{"offer_letter", "Offer letter from property vendor", "property", true},
		{"title_docs", "Property title documents", "property", true},
		{"building_plan", "Approved building plan", "property", true},
		{"valuation_report", "Valuation report from bank-approved valuer", "property", true},
		{"direct_debit_mandate", "Executed direct debit mandate and undated cheques", "banking", true},
		{"gsi_form", "Executed Global Standing Instruction (GSI) form", "banking", true},
	})
	// Indicative pre-approval fees from Infinity Trust staff checklist (valuation to be advised).
	_, _ = db.ExecContext(ctx, `
		UPDATE mortgage_products
		SET processing_fee = 20000, legal_fee = 60000, valuation_fee = NULL, updated_at = NOW()
		WHERE id = $1::uuid
	`, infinityProduct)

	// ── Link demo lender user to Stanbic IBTC ────────────────────────────────
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
	status, source, sourceURL, vstatus string, verifiedAt *time.Time) error {
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
			status=$14, source=NULLIF($15,''), source_url=NULLIF($16,''), verification_status=$17::verification_status, last_verified_at=$18, country_code=$19, updated_at=NOW()
			WHERE id=$1::uuid`, *id, desc, mtype, minLoan, maxLoan, rate, rateMin, rateMax, rateType, tenor, equity, minIncome, maxAge, status, source, sourceURL, vstatus, verified, country)
		return err
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO mortgage_products (
			lender_id, name, description, mortgage_type, min_loan_amount, max_loan_amount,
			interest_rate, interest_rate_min, interest_rate_max, interest_rate_type, max_tenor_years, min_equity_pct, min_income, max_age,
			status, source, source_url, verification_status, last_verified_at, country_code
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NULLIF($16,''),NULLIF($17,''),$18::verification_status,$19,$20)
		RETURNING id::text
	`, lenderID, name, desc, mtype, minLoan, maxLoan, rate, rateMin, rateMax, rateType, tenor, equity, minIncome, maxAge, status, source, sourceURL, vstatus, verified, country).Scan(id)
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
