package mortgages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Lender struct {
	ID                 uuid.UUID  `json:"id"`
	CountryCode        string     `json:"country_code"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Website            *string    `json:"website,omitempty"`
	Status             string     `json:"status"`
	VerificationStatus string     `json:"verification_status"`
	LastVerifiedAt     *time.Time `json:"last_verified_at,omitempty"`
}

type Product struct {
	ID                        uuid.UUID         `json:"id"`
	CountryCode               string            `json:"country_code"`
	CurrencyCode              string            `json:"currency_code"`
	LenderID                  uuid.UUID         `json:"lender_id"`
	LenderName                string            `json:"lender_name"`
	Name                      string            `json:"name"`
	Description               string            `json:"description"`
	MortgageType              string            `json:"mortgage_type"`
	MinLoanAmount             *float64          `json:"min_loan_amount"`
	MaxLoanAmount             *float64          `json:"max_loan_amount"`
	MinIncome                 *float64          `json:"min_income"`
	MaxAge                    *int              `json:"max_age"`
	MaxTenorYears             *int              `json:"max_tenor_years"`
	MinEquityPct              *float64          `json:"min_equity_pct"`
	InterestRate              *float64          `json:"interest_rate"`
	InterestRateType          string            `json:"interest_rate_type"`
	RepaymentFrequency        string            `json:"repayment_frequency"`
	ProcessingFee             *float64          `json:"processing_fee"`
	ValuationFee              *float64          `json:"valuation_fee"`
	LegalFee                  *float64          `json:"legal_fee"`
	InsuranceRequirements     *string           `json:"insurance_requirements,omitempty"`
	AcceptableEmploymentTypes json.RawMessage   `json:"acceptable_employment_types"`
	PropertyRequirements      *string           `json:"property_requirements,omitempty"`
	Status                    string            `json:"status"`
	Source                    *string           `json:"source,omitempty"`
	SourceURL                 *string           `json:"source_url,omitempty"`
	VerificationStatus        string            `json:"verification_status"`
	LastVerifiedAt            *time.Time        `json:"last_verified_at,omitempty"`
	Documents                 []ProductDocument `json:"documents,omitempty"`
	Rules                     []ProductRule     `json:"rules,omitempty"`
}

type ProductDocument struct {
	ID               uuid.UUID `json:"id"`
	DocumentTypeCode string    `json:"document_type_code"`
	Label            string    `json:"label"`
	Category         string    `json:"category"`
	Required         bool      `json:"required"`
	Instructions     string    `json:"instructions"`
}

type ProductRule struct {
	ID              uuid.UUID       `json:"id"`
	Field           string          `json:"field"`
	Operator        string          `json:"operator"`
	Value           json.RawMessage `json:"value"`
	Severity        string          `json:"severity"`
	MessageTemplate string          `json:"message_template"`
	SortOrder       int             `json:"sort_order"`
	Active          bool            `json:"active"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListLenders(ctx context.Context, countryCode string) ([]Lender, error) {
	q := `
		SELECT id, country_code, name, description, website, status, verification_status, last_verified_at
		FROM lenders
		WHERE deleted_at IS NULL AND status = 'active'`
	args := []any{}
	if countryCode != "" {
		q += ` AND country_code = $1`
		args = append(args, countryCode)
	}
	q += ` ORDER BY name`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Lender
	for rows.Next() {
		var l Lender
		var website sql.NullString
		var verified sql.NullTime
		if err := rows.Scan(&l.ID, &l.CountryCode, &l.Name, &l.Description, &website, &l.Status, &l.VerificationStatus, &verified); err != nil {
			return nil, err
		}
		if website.Valid {
			l.Website = &website.String
		}
		if verified.Valid {
			t := verified.Time
			l.LastVerifiedAt = &t
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Service) ListProducts(ctx context.Context, countryCode string) ([]Product, error) {
	q := `
		SELECT p.id, p.country_code, c.currency_code, p.lender_id, l.name, p.name, p.description, p.mortgage_type,
			p.min_loan_amount, p.max_loan_amount, p.min_income, p.max_age, p.max_tenor_years,
			p.min_equity_pct, p.interest_rate, p.interest_rate_type, p.repayment_frequency,
			p.processing_fee, p.valuation_fee, p.legal_fee, p.insurance_requirements,
			p.acceptable_employment_types, p.property_requirements, p.status,
			p.source, p.source_url, p.verification_status, p.last_verified_at
		FROM mortgage_products p
		JOIN lenders l ON l.id = p.lender_id
		JOIN countries c ON c.code = p.country_code
		WHERE p.deleted_at IS NULL AND p.status = 'active' AND l.deleted_at IS NULL`
	args := []any{}
	if countryCode != "" {
		q += ` AND p.country_code = $1`
		args = append(args, countryCode)
	}
	q += ` ORDER BY p.interest_rate NULLS LAST, p.name`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.country_code, c.currency_code, p.lender_id, l.name, p.name, p.description, p.mortgage_type,
			p.min_loan_amount, p.max_loan_amount, p.min_income, p.max_age, p.max_tenor_years,
			p.min_equity_pct, p.interest_rate, p.interest_rate_type, p.repayment_frequency,
			p.processing_fee, p.valuation_fee, p.legal_fee, p.insurance_requirements,
			p.acceptable_employment_types, p.property_requirements, p.status,
			p.source, p.source_url, p.verification_status, p.last_verified_at
		FROM mortgage_products p
		JOIN lenders l ON l.id = p.lender_id
		JOIN countries c ON c.code = p.country_code
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, id)
	p, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	docs, err := s.listDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	rules, err := s.listRules(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Documents = docs
	p.Rules = rules
	return &p, nil
}

func (s *Service) listDocuments(ctx context.Context, productID uuid.UUID) ([]ProductDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_type_code, label, category, required, instructions
		FROM mortgage_product_documents
		WHERE product_id = $1
		ORDER BY category, label
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductDocument
	for rows.Next() {
		var d ProductDocument
		if err := rows.Scan(&d.ID, &d.DocumentTypeCode, &d.Label, &d.Category, &d.Required, &d.Instructions); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Service) listRules(ctx context.Context, productID uuid.UUID) ([]ProductRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, field, operator, value, severity, message_template, sort_order, active
		FROM mortgage_rules
		WHERE product_id = $1 AND active = TRUE
		ORDER BY sort_order, field
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductRule
	for rows.Next() {
		var r ProductRule
		if err := rows.Scan(&r.ID, &r.Field, &r.Operator, &r.Value, &r.Severity, &r.MessageTemplate, &r.SortOrder, &r.Active); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) RulesForEngine(ctx context.Context, productID uuid.UUID) ([]ProductRule, error) {
	return s.listRules(ctx, productID)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanProduct(row scannable) (Product, error) {
	var p Product
	var (
		minLoan, maxLoan, minIncome, minEquity, rate sql.NullFloat64
		proc, val, legal                             sql.NullFloat64
		maxAge, maxTenor                             sql.NullInt64
		ins, prop, source, sourceURL                 sql.NullString
		verified                                     sql.NullTime
		empTypes                                     []byte
	)
	err := row.Scan(
		&p.ID, &p.CountryCode, &p.CurrencyCode, &p.LenderID, &p.LenderName, &p.Name, &p.Description, &p.MortgageType,
		&minLoan, &maxLoan, &minIncome, &maxAge, &maxTenor,
		&minEquity, &rate, &p.InterestRateType, &p.RepaymentFrequency,
		&proc, &val, &legal, &ins,
		&empTypes, &prop, &p.Status,
		&source, &sourceURL, &p.VerificationStatus, &verified,
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
	p.InsuranceRequirements = nullS(ins)
	p.PropertyRequirements = nullS(prop)
	p.Source = nullS(source)
	p.SourceURL = nullS(sourceURL)
	if len(empTypes) == 0 {
		p.AcceptableEmploymentTypes = json.RawMessage("[]")
	} else {
		p.AcceptableEmploymentTypes = empTypes
	}
	if verified.Valid {
		t := verified.Time
		p.LastVerifiedAt = &t
	}
	return p, nil
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
