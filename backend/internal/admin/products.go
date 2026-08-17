package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type adminLender struct {
	ID          string  `json:"id"`
	CountryCode string  `json:"country_code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Website     *string `json:"website,omitempty"`
	Status      string  `json:"status"`
}

type adminProduct struct {
	ID                 string     `json:"id"`
	CountryCode        string     `json:"country_code"`
	CurrencyCode       string     `json:"currency_code"`
	LenderID           string     `json:"lender_id"`
	LenderName         string     `json:"lender_name"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	MortgageType       string     `json:"mortgage_type"`
	MinLoanAmount      *float64   `json:"min_loan_amount"`
	MaxLoanAmount      *float64   `json:"max_loan_amount"`
	MinIncome          *float64   `json:"min_income"`
	MaxAge             *int       `json:"max_age"`
	MaxTenorYears      *int       `json:"max_tenor_years"`
	MinEquityPct       *float64   `json:"min_equity_pct"`
	InterestRate       *float64   `json:"interest_rate"`
	InterestRateType   string     `json:"interest_rate_type"`
	ProcessingFee      *float64   `json:"processing_fee"`
	ValuationFee       *float64   `json:"valuation_fee"`
	LegalFee           *float64   `json:"legal_fee"`
	Status             string     `json:"status"`
	Source             *string    `json:"source,omitempty"`
	SourceURL          *string    `json:"source_url,omitempty"`
	VerificationStatus string     `json:"verification_status"`
	LastVerifiedAt     *time.Time `json:"last_verified_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type productWrite struct {
	LenderID           string   `json:"lender_id" binding:"required"`
	CountryCode        string   `json:"country_code" binding:"required"`
	Name               string   `json:"name" binding:"required,min=2"`
	Description        string   `json:"description"`
	MortgageType       string   `json:"mortgage_type" binding:"required"`
	MinLoanAmount      *float64 `json:"min_loan_amount"`
	MaxLoanAmount      *float64 `json:"max_loan_amount"`
	MinIncome          *float64 `json:"min_income"`
	MaxAge             *int     `json:"max_age"`
	MaxTenorYears      *int     `json:"max_tenor_years"`
	MinEquityPct       *float64 `json:"min_equity_pct"`
	InterestRate       *float64 `json:"interest_rate"`
	InterestRateType   string   `json:"interest_rate_type"`
	ProcessingFee      *float64 `json:"processing_fee"`
	ValuationFee       *float64 `json:"valuation_fee"`
	LegalFee           *float64 `json:"legal_fee"`
	Status             string   `json:"status"`
	Source             string   `json:"source"`
	SourceURL          string   `json:"source_url"`
	VerificationStatus string   `json:"verification_status"`
	SyncRules          *bool    `json:"sync_rules"`
}

type lenderWrite struct {
	Name        string `json:"name" binding:"required,min=2"`
	CountryCode string `json:"country_code" binding:"required"`
	Description string `json:"description"`
	Website     string `json:"website"`
}

const productSelect = `
	SELECT p.id::text, p.country_code, c.currency_code, p.lender_id::text, l.name, p.name, p.description, p.mortgage_type,
		p.min_loan_amount, p.max_loan_amount, p.min_income, p.max_age, p.max_tenor_years,
		p.min_equity_pct, p.interest_rate, p.interest_rate_type,
		p.processing_fee, p.valuation_fee, p.legal_fee, p.status,
		p.source, p.source_url, p.verification_status, p.last_verified_at, p.updated_at
	FROM mortgage_products p
	JOIN lenders l ON l.id = p.lender_id
	JOIN countries c ON c.code = p.country_code
`

func (h *Handler) ListLenders(c *gin.Context) {
	q := `
		SELECT id::text, country_code, name, description, website, status
		FROM lenders WHERE deleted_at IS NULL`
	args := []any{}
	if country := strings.TrimSpace(c.Query("country")); country != "" {
		q += ` AND country_code = $1`
		args = append(args, country)
	}
	q += ` ORDER BY name`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		httpx.Internal(c, "Could not load lenders.")
		return
	}
	defer rows.Close()
	out := []adminLender{}
	for rows.Next() {
		var it adminLender
		var website sql.NullString
		if err := rows.Scan(&it.ID, &it.CountryCode, &it.Name, &it.Description, &website, &it.Status); err != nil {
			httpx.Internal(c, "Could not load lenders.")
			return
		}
		if website.Valid {
			it.Website = &website.String
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"lenders": out})
}

func (h *Handler) CreateLender(c *gin.Context) {
	var in lenderWrite
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Lender name and country are required.")
		return
	}
	if err := h.requireCountry(c.Request.Context(), in.CountryCode); err != nil {
		httpx.BadRequest(c, "Unknown country.")
		return
	}
	var id string
	err := h.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO lenders (name, description, website, country_code, status, verification_status)
		VALUES ($1, $2, NULLIF($3,''), $4, 'active', 'needs_verification')
		RETURNING id::text
	`, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), strings.TrimSpace(in.Website), strings.ToUpper(strings.TrimSpace(in.CountryCode))).Scan(&id)
	if err != nil {
		httpx.Internal(c, "Could not create lender.")
		return
	}
	item, err := h.getLender(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Lender created but could not be loaded.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"lender": item})
}

func (h *Handler) ListProducts(c *gin.Context) {
	q := productSelect + ` WHERE p.deleted_at IS NULL`
	args := []any{}
	if country := strings.TrimSpace(c.Query("country")); country != "" {
		q += ` AND p.country_code = $1`
		args = append(args, country)
	}
	q += ` ORDER BY p.updated_at DESC`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		httpx.Internal(c, "Could not load products.")
		return
	}
	defer rows.Close()
	out := []adminProduct{}
	for rows.Next() {
		p, err := scanAdminProduct(rows)
		if err != nil {
			httpx.Internal(c, "Could not load products.")
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"products": out})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var in productWrite
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Name, lender, country, and mortgage type are required.")
		return
	}
	if msg := validateProductWrite(in); msg != "" {
		httpx.BadRequest(c, msg)
		return
	}
	if err := h.requireCountry(c.Request.Context(), in.CountryCode); err != nil {
		httpx.BadRequest(c, "Unknown country.")
		return
	}
	if err := h.requireLender(c.Request.Context(), in.LenderID, in.CountryCode); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	var id string
	err := h.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO mortgage_products (
			lender_id, country_code, name, description, mortgage_type,
			min_loan_amount, max_loan_amount, min_income, max_age, max_tenor_years, min_equity_pct,
			interest_rate, interest_rate_type, processing_fee, valuation_fee, legal_fee,
			status, source, source_url, verification_status, last_verified_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, NULLIF($18,''), NULLIF($19,''), $20::verification_status,
			CASE WHEN $20::text = 'verified' THEN NOW() ELSE NULL END
		) RETURNING id::text
	`, in.LenderID, strings.ToUpper(in.CountryCode), strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.MortgageType,
		in.MinLoanAmount, in.MaxLoanAmount, in.MinIncome, in.MaxAge, in.MaxTenorYears, in.MinEquityPct,
		in.InterestRate, defaultRateType(in.InterestRateType), in.ProcessingFee, in.ValuationFee, in.LegalFee,
		defaultProductStatus(in.Status), strings.TrimSpace(in.Source), strings.TrimSpace(in.SourceURL), defaultVerification(in.VerificationStatus),
	).Scan(&id)
	if err != nil {
		httpx.Internal(c, "Could not create product.")
		return
	}
	if in.SyncRules == nil || *in.SyncRules {
		if err := h.syncDerivedRules(c.Request.Context(), id, in, true); err != nil {
			httpx.Internal(c, "Product saved but eligibility rules could not be updated.")
			return
		}
	}
	if err := h.ensureDefaultDocuments(c.Request.Context(), id); err != nil {
		httpx.Internal(c, "Product saved but the default document list could not be attached.")
		return
	}
	p, err := h.getProduct(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Product created but could not be loaded.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": p})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown product.")
		return
	}
	if _, err := h.getProduct(c.Request.Context(), id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(c, "Product not found.")
		return
	} else if err != nil {
		httpx.Internal(c, "Could not update product.")
		return
	}
	var in productWrite
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Name, lender, country, and mortgage type are required.")
		return
	}
	if msg := validateProductWrite(in); msg != "" {
		httpx.BadRequest(c, msg)
		return
	}
	if err := h.requireCountry(c.Request.Context(), in.CountryCode); err != nil {
		httpx.BadRequest(c, "Unknown country.")
		return
	}
	if err := h.requireLender(c.Request.Context(), in.LenderID, in.CountryCode); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	_, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE mortgage_products SET
			lender_id=$2::uuid, country_code=$3, name=$4, description=$5, mortgage_type=$6,
			min_loan_amount=$7, max_loan_amount=$8, min_income=$9, max_age=$10, max_tenor_years=$11, min_equity_pct=$12,
			interest_rate=$13, interest_rate_type=$14, processing_fee=$15, valuation_fee=$16, legal_fee=$17,
			status=$18, source=NULLIF($19,''), source_url=NULLIF($20,''),
			verification_status=$21::verification_status,
			last_verified_at = CASE
				WHEN $21::text = 'verified' THEN NOW()
				ELSE last_verified_at
			END,
			updated_at=NOW()
		WHERE id=$1::uuid AND deleted_at IS NULL
	`, id, in.LenderID, strings.ToUpper(in.CountryCode), strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.MortgageType,
		in.MinLoanAmount, in.MaxLoanAmount, in.MinIncome, in.MaxAge, in.MaxTenorYears, in.MinEquityPct,
		in.InterestRate, defaultRateType(in.InterestRateType), in.ProcessingFee, in.ValuationFee, in.LegalFee,
		defaultProductStatus(in.Status), strings.TrimSpace(in.Source), strings.TrimSpace(in.SourceURL), defaultVerification(in.VerificationStatus),
	)
	if err != nil {
		httpx.Internal(c, "Could not update product.")
		return
	}
	if in.SyncRules == nil || *in.SyncRules {
		if err := h.syncDerivedRules(c.Request.Context(), id, in, false); err != nil {
			httpx.Internal(c, "Product saved but eligibility rules could not be updated.")
			return
		}
	}
	p, err := h.getProduct(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Could not load updated product.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown product.")
		return
	}
	res, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE mortgage_products SET deleted_at=NOW(), status='inactive', updated_at=NOW()
		WHERE id=$1::uuid AND deleted_at IS NULL
	`, id)
	if err != nil {
		httpx.Internal(c, "Could not remove product.")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpx.NotFound(c, "Product not found.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) getProduct(ctx context.Context, id string) (*adminProduct, error) {
	row := h.db.QueryRowContext(ctx, productSelect+` WHERE p.id=$1::uuid AND p.deleted_at IS NULL`, id)
	p, err := scanAdminProduct(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (h *Handler) getLender(ctx context.Context, id string) (*adminLender, error) {
	var it adminLender
	var website sql.NullString
	err := h.db.QueryRowContext(ctx, `
		SELECT id::text, country_code, name, description, website, status
		FROM lenders WHERE id=$1::uuid AND deleted_at IS NULL
	`, id).Scan(&it.ID, &it.CountryCode, &it.Name, &it.Description, &website, &it.Status)
	if err != nil {
		return nil, err
	}
	if website.Valid {
		it.Website = &website.String
	}
	return &it, nil
}

func (h *Handler) requireCountry(ctx context.Context, code string) error {
	var exists string
	return h.db.QueryRowContext(ctx, `SELECT code FROM countries WHERE code=$1`, strings.ToUpper(strings.TrimSpace(code))).Scan(&exists)
}

func (h *Handler) requireLender(ctx context.Context, id, country string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("Choose a lender.")
	}
	var lenderCountry string
	err := h.db.QueryRowContext(ctx, `
		SELECT country_code FROM lenders WHERE id=$1::uuid AND deleted_at IS NULL
	`, id).Scan(&lenderCountry)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("Choose a lender.")
	}
	if err != nil {
		return errors.New("Could not check lender.")
	}
	if !strings.EqualFold(lenderCountry, country) {
		return errors.New("That lender belongs to a different country.")
	}
	return nil
}

func (h *Handler) syncDerivedRules(ctx context.Context, productID string, in productWrite, includeDefaults bool) error {
	type derived struct {
		field, op, severity, msg string
		value                    any
	}
	wanted := []derived{}
	if in.MinIncome != nil {
		wanted = append(wanted, derived{"monthly_income", "gte", "hard", "This product usually needs enough income to keep repayments affordable.", *in.MinIncome})
	}
	if in.MaxAge != nil {
		wanted = append(wanted, derived{"age", "lte", "hard", "You should typically complete repayment before age {value}.", *in.MaxAge})
	}
	if in.MinEquityPct != nil {
		wanted = append(wanted, derived{"equity_pct", "gte", "hard", "Minimum equity contribution is typically {value}%.", *in.MinEquityPct})
	}
	if in.MinLoanAmount != nil {
		wanted = append(wanted, derived{"loan_amount", "gte", "hard", "Minimum loan amount is typically {value}.", *in.MinLoanAmount})
	}
	if in.MaxLoanAmount != nil {
		wanted = append(wanted, derived{"loan_amount", "lte", "hard", "Maximum loan amount is typically {value}.", *in.MaxLoanAmount})
	}

	keep := map[string]bool{}
	for i, r := range wanted {
		keep[r.field+"|"+r.op] = true
		raw, _ := json.Marshal(r.value)
		res, err := h.db.ExecContext(ctx, `
			UPDATE mortgage_rules
			SET value=$4::jsonb, severity=$5, message_template=$6, sort_order=$7, active=TRUE, updated_at=NOW()
			WHERE product_id=$1::uuid AND field=$2 AND operator=$3
		`, productID, r.field, r.op, string(raw), r.severity, r.msg, i)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			if _, err := h.db.ExecContext(ctx, `
				INSERT INTO mortgage_rules (product_id, field, operator, value_type, value, severity, message_template, sort_order)
				VALUES ($1::uuid,$2,$3,'number',$4::jsonb,$5,$6,$7)
			`, productID, r.field, r.op, string(raw), r.severity, r.msg, i); err != nil {
				return err
			}
		}
	}
	derivedKeys := [][2]string{
		{"monthly_income", "gte"}, {"age", "lte"}, {"equity_pct", "gte"}, {"loan_amount", "gte"}, {"loan_amount", "lte"},
	}
	for _, k := range derivedKeys {
		if keep[k[0]+"|"+k[1]] {
			continue
		}
		if _, err := h.db.ExecContext(ctx, `
			UPDATE mortgage_rules SET active=FALSE, updated_at=NOW()
			WHERE product_id=$1::uuid AND field=$2 AND operator=$3
		`, productID, k[0], k[1]); err != nil {
			return err
		}
	}

	if includeDefaults {
		defaults := []struct {
			field, op, valType, val, severity, msg string
			sort                                   int
		}{
			{"employment_type", "in", "list", `["salaried","civil_servant"]`, "hard", "HomeGauge’s automated review currently supports salary-account workers.", 20},
			{"salary_months", "gte", "number", "6", "hard", "We look for about 6 months of clear salary credits on one account.", 21},
			{"iti_pct", "lte", "number", "35", "soft", "Keeping repayments near or below {value}% of income improves fit.", 22},
		}
		for _, d := range defaults {
			var exists int
			_ = h.db.QueryRowContext(ctx, `
				SELECT 1 FROM mortgage_rules WHERE product_id=$1::uuid AND field=$2 AND operator=$3 LIMIT 1
			`, productID, d.field, d.op).Scan(&exists)
			if exists == 1 {
				continue
			}
			if _, err := h.db.ExecContext(ctx, `
				INSERT INTO mortgage_rules (product_id, field, operator, value_type, value, severity, message_template, sort_order)
				VALUES ($1::uuid,$2,$3,$4,$5::jsonb,$6,$7,$8)
			`, productID, d.field, d.op, d.valType, d.val, d.severity, d.msg, d.sort); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) ensureDefaultDocuments(ctx context.Context, productID string) error {
	var n int
	if err := h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mortgage_product_documents WHERE product_id=$1::uuid`, productID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	codes := []string{"valid_id", "payslips_3m", "employment_letter", "salary_statements_6m", "offer_letter", "title_docs"}
	for _, code := range codes {
		if _, err := h.db.ExecContext(ctx, `
			INSERT INTO mortgage_product_documents (product_id, document_type_code, label, category, required, instructions)
			SELECT $1::uuid, code, label, category, TRUE, ''
			FROM document_types WHERE code=$2
		`, productID, code); err != nil {
			return err
		}
	}
	return nil
}

func scanAdminProduct(row scannable) (adminProduct, error) {
	var p adminProduct
	var (
		minLoan, maxLoan, minIncome, minEquity, rate sql.NullFloat64
		proc, val, legal                             sql.NullFloat64
		maxAge, maxTenor                             sql.NullInt64
		source, sourceURL                            sql.NullString
		verified                                     sql.NullTime
	)
	err := row.Scan(
		&p.ID, &p.CountryCode, &p.CurrencyCode, &p.LenderID, &p.LenderName, &p.Name, &p.Description, &p.MortgageType,
		&minLoan, &maxLoan, &minIncome, &maxAge, &maxTenor,
		&minEquity, &rate, &p.InterestRateType,
		&proc, &val, &legal, &p.Status,
		&source, &sourceURL, &p.VerificationStatus, &verified, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.MinLoanAmount = nullF(minLoan)
	p.MaxLoanAmount = nullF(maxLoan)
	p.MinIncome = nullF(minIncome)
	p.MinEquityPct = nullF(minEquity)
	p.InterestRate = nullF(rate)
	p.ProcessingFee = nullF(proc)
	p.ValuationFee = nullF(val)
	p.LegalFee = nullF(legal)
	if maxAge.Valid {
		v := int(maxAge.Int64)
		p.MaxAge = &v
	}
	if maxTenor.Valid {
		v := int(maxTenor.Int64)
		p.MaxTenorYears = &v
	}
	p.Source = nullS(source)
	p.SourceURL = nullS(sourceURL)
	if verified.Valid {
		t := verified.Time
		p.LastVerifiedAt = &t
	}
	return p, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func validateProductWrite(in productWrite) string {
	if _, err := uuid.Parse(in.LenderID); err != nil {
		return "Choose a lender."
	}
	if strings.TrimSpace(in.CountryCode) == "" {
		return "Choose a country."
	}
	if !validMortgageType(in.MortgageType) {
		return "Mortgage type must be nhf, mreif, commercial, scheme, or other."
	}
	if in.InterestRateType != "" && !validRateType(in.InterestRateType) {
		return "Interest rate type must be fixed or variable."
	}
	if in.Status != "" && !validProductStatus(in.Status) {
		return "Status must be active or inactive."
	}
	if in.VerificationStatus != "" && !validVerification(in.VerificationStatus) {
		return "Verification must be verified, needs_verification, or expired."
	}
	if in.MinLoanAmount != nil && in.MaxLoanAmount != nil && *in.MinLoanAmount > *in.MaxLoanAmount {
		return "Minimum loan cannot be greater than maximum loan."
	}
	return ""
}

func validMortgageType(v string) bool {
	switch v {
	case "nhf", "mreif", "commercial", "scheme", "other":
		return true
	default:
		return false
	}
}

func validRateType(v string) bool {
	return v == "fixed" || v == "variable"
}

func validProductStatus(v string) bool {
	return v == "active" || v == "inactive"
}

func validVerification(v string) bool {
	return v == "verified" || v == "needs_verification" || v == "expired"
}

func defaultRateType(v string) string {
	if validRateType(v) {
		return v
	}
	return "fixed"
}

func defaultProductStatus(v string) string {
	if validProductStatus(v) {
		return v
	}
	return "active"
}

func defaultVerification(v string) string {
	if validVerification(v) {
		return v
	}
	return "needs_verification"
}

func nullF(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

func nullS(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}
