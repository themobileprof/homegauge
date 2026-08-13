package eligibility

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/calculator"
	"github.com/homegauge/homegauge/backend/internal/mortgages"
	"github.com/homegauge/homegauge/backend/internal/readiness"
)

var (
	ErrAssessmentNotFound = errors.New("assessment not found")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidStep        = errors.New("invalid assessment step")
)

type Service struct {
	db        *sql.DB
	mortgages *mortgages.Service
	defaultITI float64
}

func NewService(db *sql.DB, m *mortgages.Service, defaultITI float64) *Service {
	return &Service{db: db, mortgages: m, defaultITI: defaultITI}
}

type AssessmentInput struct {
	CountryCode          string  `json:"country_code"`
	FullName             string  `json:"full_name"`
	DateOfBirth          string  `json:"date_of_birth"` // YYYY-MM-DD
	Age                  int     `json:"age"`
	StateOfResidence     string  `json:"state_of_residence"` // region / state / county within country
	ResidencyType        string  `json:"residency_type"`
	MaritalStatus        string  `json:"marital_status"`
	EmploymentType       string  `json:"employment_type"`
	EmployerName         string  `json:"employer_name"`
	YearsEmployed        float64 `json:"years_employed"`
	MonthlyNetIncome     float64 `json:"monthly_net_income"`
	OtherMonthlyIncome   float64 `json:"other_monthly_income"`
	MonthlyExpenses      float64 `json:"monthly_expenses"`
	ExistingDebt         float64 `json:"existing_debt_repayments"`
	AvailableDeposit     float64 `json:"available_deposit"`
	DesiredPropertyPrice float64 `json:"desired_property_price"`
	DesiredLoanAmount    float64 `json:"desired_loan_amount"`
	PreferredTenorYears  int     `json:"preferred_tenor_years"`
	SalaryMonths         int     `json:"salary_months"` // self-declared until statement AI
	NHFContributorMonths int     `json:"nhf_contributor_months"` // scheme months (NG NHF); ignored for markets without that rule
	WillingToDomicile    bool    `json:"willing_to_domicile_salary"`
}

type Assessment struct {
	ID             uuid.UUID        `json:"id"`
	UserID         uuid.UUID        `json:"user_id"`
	Status         string           `json:"status"`
	InputSnapshot  AssessmentInput  `json:"input_snapshot"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	Results        []ProductOutcome `json:"results,omitempty"`
	Readiness      *readiness.Score `json:"readiness,omitempty"`
	BestFitProduct *uuid.UUID       `json:"best_fit_product_id,omitempty"`
	BestFitWhy     string           `json:"best_fit_why,omitempty"`
}

type ProductOutcome struct {
	ProductID                  uuid.UUID       `json:"product_id"`
	ProductName                string          `json:"product_name"`
	LenderName                 string          `json:"lender_name"`
	Outcome                    Outcome         `json:"outcome"`
	Explanation                string          `json:"explanation"`
	Detail                     json.RawMessage `json:"detail"`
	EstimatedMonthlyRepayment  *float64        `json:"estimated_monthly_repayment,omitempty"`
	InterestRate               *float64        `json:"interest_rate,omitempty"`
	MinEquityPct               *float64        `json:"min_equity_pct,omitempty"`
	VerificationStatus         string          `json:"verification_status"`
	LastVerifiedAt             *time.Time      `json:"last_verified_at,omitempty"`
}

func (s *Service) Start(ctx context.Context, userID uuid.UUID) (*Assessment, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO eligibility_assessments (user_id, status, input_snapshot)
		VALUES ($1, 'in_progress', '{}')
		RETURNING id
	`, userID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

func (s *Service) UpdateStep(ctx context.Context, userID, assessmentID uuid.UUID, patch AssessmentInput) (*Assessment, error) {
	a, err := s.Get(ctx, userID, assessmentID)
	if err != nil {
		return nil, err
	}
	if a.Status != "in_progress" {
		return nil, ErrInvalidStep
	}
	merged := mergeInput(a.InputSnapshot, patch)
	b, _ := json.Marshal(merged)
	_, err = s.db.ExecContext(ctx, `
		UPDATE eligibility_assessments
		SET input_snapshot = $2::jsonb, country_code = COALESCE(NULLIF($3,''), country_code), updated_at = NOW()
		WHERE id = $1
	`, assessmentID, string(b), merged.CountryCode)
	if err != nil {
		return nil, err
	}
	_ = s.syncProfiles(ctx, userID, merged)
	return s.Get(ctx, userID, assessmentID)
}

func (s *Service) Complete(ctx context.Context, userID, assessmentID uuid.UUID) (*Assessment, error) {
	a, err := s.Get(ctx, userID, assessmentID)
	if err != nil {
		return nil, err
	}
	if a.Status != "in_progress" {
		return a, nil
	}
	in := a.InputSnapshot
	if in.EmploymentType == "" {
		in.EmploymentType = "salaried"
	}
	if in.Age == 0 && in.DateOfBirth != "" {
		in.Age = ageFromDOB(in.DateOfBirth)
	}
	country := in.CountryCode
	if country == "" {
		country = "NG"
	}

	products, err := s.mortgages.ListProducts(ctx, country)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.ExecContext(ctx, `DELETE FROM eligibility_results WHERE assessment_id = $1`, assessmentID)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM readiness_scores WHERE assessment_id = $1`, assessmentID)

	var outcomes []ProductOutcome
	likely := 0
	var best *ProductOutcome
	minEquityForBest := 10.0
	var bestITI float64

	for _, p := range products {
		rulesRows, err := s.mortgages.RulesForEngine(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		rules := toEngineRules(rulesRows)
		evalCtx, estPay, iti := s.buildContext(in, p)
		ev := Evaluate(evalCtx, rules)
		detail, _ := json.Marshal(ev)
		var payPtr *float64
		if estPay > 0 {
			v := math.Round(estPay*100) / 100
			payPtr = &v
		}
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO eligibility_results (assessment_id, product_id, outcome, detail, explanation, estimated_monthly_repayment)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		`, assessmentID, p.ID, string(ev.Outcome), string(detail), ev.Summary, payPtr)
		if err != nil {
			return nil, err
		}
		out := ProductOutcome{
			ProductID:                 p.ID,
			ProductName:               p.Name,
			LenderName:                p.LenderName,
			Outcome:                   ev.Outcome,
			Explanation:               ev.Summary,
			Detail:                    detail,
			EstimatedMonthlyRepayment: payPtr,
			InterestRate:              p.InterestRate,
			MinEquityPct:              p.MinEquityPct,
			VerificationStatus:        p.VerificationStatus,
			LastVerifiedAt:            p.LastVerifiedAt,
		}
		outcomes = append(outcomes, out)
		if ev.Outcome == LikelyEligible || ev.Outcome == PotentiallyEligible {
			if ev.Outcome == LikelyEligible {
				likely++
			}
			if best == nil || betterFit(out, *best) {
				cp := out
				best = &cp
				if p.MinEquityPct != nil {
					minEquityForBest = *p.MinEquityPct
				}
				bestITI = iti
			}
		}
		_ = iti
	}

	salaryMonths := in.SalaryMonths
	score := readiness.Compute(readiness.Input{
		SalaryMonthsDetected: salaryMonths,
		SalaryVarianceOK:     salaryMonths >= 6,
		DeclaredSalary:       in.MonthlyNetIncome,
		Deposit:              in.AvailableDeposit,
		PropertyPrice:        in.DesiredPropertyPrice,
		MinEquityPct:         minEquityForBest,
		ITIPct:               bestITI,
		YearsEmployed:        in.YearsEmployed,
		DocsReadyPct:         0,
		LikelyEligibleCount:  likely,
		ProductCount:         len(products),
	})
	compJSON, _ := json.Marshal(score.Components)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO readiness_scores (assessment_id, user_id, total_score, components, narrative)
		VALUES ($1, $2, $3, $4::jsonb, $5)
	`, assessmentID, userID, score.Total, string(compJSON), score.Narrative)
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE eligibility_assessments
		SET status = 'completed', completed_at = NOW(), updated_at = NOW(), input_snapshot = $2::jsonb
		WHERE id = $1
	`, assessmentID, mustJSON(in))
	if err != nil {
		return nil, err
	}

	// Ensure an application case exists for concierge later
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO mortgage_applications (user_id, assessment_id, status, next_action_text)
		SELECT $1, $2, 'ASSESSMENT_COMPLETED', 'Review your results and prepare your salary account statements.'
		WHERE NOT EXISTS (
			SELECT 1 FROM mortgage_applications WHERE assessment_id = $2
		)
	`, userID, assessmentID)

	return s.Get(ctx, userID, assessmentID)
}

func betterFit(a, b ProductOutcome) bool {
	rank := func(o Outcome) int {
		switch o {
		case LikelyEligible:
			return 0
		case PotentiallyEligible:
			return 1
		case MayRequireReview:
			return 2
		default:
			return 9
		}
	}
	if rank(a.Outcome) != rank(b.Outcome) {
		return rank(a.Outcome) < rank(b.Outcome)
	}
	ar, br := 99.0, 99.0
	if a.InterestRate != nil {
		ar = *a.InterestRate
	}
	if b.InterestRate != nil {
		br = *b.InterestRate
	}
	return ar < br
}

func (s *Service) buildContext(in AssessmentInput, p mortgages.Product) (Context, float64, float64) {
	loan := in.DesiredLoanAmount
	if loan <= 0 && in.DesiredPropertyPrice > 0 {
		loan = math.Max(0, in.DesiredPropertyPrice-in.AvailableDeposit)
	}
	equityPct := 0.0
	if in.DesiredPropertyPrice > 0 {
		equityPct = (in.AvailableDeposit / in.DesiredPropertyPrice) * 100
	}
	tenor := in.PreferredTenorYears
	if tenor <= 0 && p.MaxTenorYears != nil {
		tenor = *p.MaxTenorYears
	}
	if tenor <= 0 {
		tenor = 20
	}
	rate := 0.0
	if p.InterestRate != nil {
		rate = *p.InterestRate
	}
	est := calculator.Affordability(calculator.Input{
		PropertyPrice:       in.DesiredPropertyPrice,
		Deposit:             in.AvailableDeposit,
		LoanAmount:          loan,
		AnnualInterestRate:  rate,
		TenorYears:          tenor,
		MonthlyIncome:       in.MonthlyNetIncome,
		ExistingMonthlyDebt: in.ExistingDebt,
	})

	ctx := Context{
		"monthly_income":         in.MonthlyNetIncome,
		"age":                    float64(in.Age),
		"employment_type":        in.EmploymentType,
		"years_employed":         in.YearsEmployed,
		"equity_pct":             equityPct,
		"loan_amount":            loan,
		"salary_months":          float64(in.SalaryMonths),
		"nhf_contributor_months": float64(in.NHFContributorMonths),
		"iti_pct":                est.InstallmentToIncomePct,
		"state_of_residence":     in.StateOfResidence,
	}
	return ctx, est.MonthlyRepayment, est.InstallmentToIncomePct
}

func (s *Service) Get(ctx context.Context, userID, assessmentID uuid.UUID) (*Assessment, error) {
	var (
		a          Assessment
		snapshot   []byte
		completed  sql.NullTime
		status     string
		started    time.Time
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, status, input_snapshot, started_at, completed_at
		FROM eligibility_assessments WHERE id = $1
	`, assessmentID).Scan(&a.ID, &a.UserID, &status, &snapshot, &started, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssessmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, ErrForbidden
	}
	a.Status = status
	a.StartedAt = started
	if completed.Valid {
		t := completed.Time
		a.CompletedAt = &t
	}
	_ = json.Unmarshal(snapshot, &a.InputSnapshot)

	if status == "completed" {
		results, err := s.loadResults(ctx, assessmentID)
		if err != nil {
			return nil, err
		}
		a.Results = results
		score, err := s.loadReadiness(ctx, assessmentID)
		if err != nil {
			return nil, err
		}
		a.Readiness = score
		if best, why := pickBest(results); best != nil {
			a.BestFitProduct = &best.ProductID
			a.BestFitWhy = why
		}
	}
	return &a, nil
}

func (s *Service) LatestForUser(ctx context.Context, userID uuid.UUID) (*Assessment, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM eligibility_assessments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssessmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

func (s *Service) loadResults(ctx context.Context, assessmentID uuid.UUID) ([]ProductOutcome, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.product_id, p.name, l.name, r.outcome, r.explanation, r.detail, r.estimated_monthly_repayment,
			p.interest_rate, p.min_equity_pct, p.verification_status, p.last_verified_at
		FROM eligibility_results r
		JOIN mortgage_products p ON p.id = r.product_id
		JOIN lenders l ON l.id = p.lender_id
		WHERE r.assessment_id = $1
		ORDER BY
			CASE r.outcome
				WHEN 'likely_eligible' THEN 0
				WHEN 'potentially_eligible' THEN 1
				WHEN 'may_require_review' THEN 2
				WHEN 'more_info_required' THEN 3
				ELSE 4
			END,
			p.interest_rate NULLS LAST
	`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductOutcome
	for rows.Next() {
		var o ProductOutcome
		var pay, rate, equity sql.NullFloat64
		var verified sql.NullTime
		var detail []byte
		if err := rows.Scan(&o.ProductID, &o.ProductName, &o.LenderName, &o.Outcome, &o.Explanation, &detail, &pay, &rate, &equity, &o.VerificationStatus, &verified); err != nil {
			return nil, err
		}
		o.Detail = detail
		if pay.Valid {
			v := pay.Float64
			o.EstimatedMonthlyRepayment = &v
		}
		if rate.Valid {
			v := rate.Float64
			o.InterestRate = &v
		}
		if equity.Valid {
			v := equity.Float64
			o.MinEquityPct = &v
		}
		if verified.Valid {
			t := verified.Time
			o.LastVerifiedAt = &t
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Service) loadReadiness(ctx context.Context, assessmentID uuid.UUID) (*readiness.Score, error) {
	var total int
	var narrative string
	var components []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT total_score, components, narrative
		FROM readiness_scores WHERE assessment_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, assessmentID).Scan(&total, &components, &narrative)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var comps []readiness.Component
	_ = json.Unmarshal(components, &comps)
	return &readiness.Score{Total: total, Components: comps, Narrative: narrative}, nil
}

func pickBest(results []ProductOutcome) (*ProductOutcome, string) {
	var best *ProductOutcome
	for i := range results {
		r := results[i]
		if r.Outcome != LikelyEligible && r.Outcome != PotentiallyEligible {
			continue
		}
		if best == nil || betterFit(r, *best) {
			cp := r
			best = &cp
		}
	}
	if best == nil {
		return nil, ""
	}
	why := fmt.Sprintf(
		"Best fit among products you may qualify for: %s (%s), because it has a stronger eligibility match and a comparatively lower stated interest rate. This is not a bank approval.",
		best.ProductName, best.LenderName,
	)
	return best, why
}

func toEngineRules(rows []mortgages.ProductRule) []Rule {
	out := make([]Rule, 0, len(rows))
	for _, r := range rows {
		var val any
		_ = json.Unmarshal(r.Value, &val)
		out = append(out, Rule{
			Field: r.Field, Operator: r.Operator, Value: val,
			Severity: r.Severity, MessageTemplate: r.MessageTemplate,
		})
	}
	return out
}

func mergeInput(base, patch AssessmentInput) AssessmentInput {
	out := base
	if patch.CountryCode != "" {
		out.CountryCode = patch.CountryCode
	}
	if patch.FullName != "" {
		out.FullName = patch.FullName
	}
	if patch.DateOfBirth != "" {
		out.DateOfBirth = patch.DateOfBirth
	}
	if patch.Age != 0 {
		out.Age = patch.Age
	}
	if patch.StateOfResidence != "" {
		out.StateOfResidence = patch.StateOfResidence
	}
	if patch.ResidencyType != "" {
		out.ResidencyType = patch.ResidencyType
	}
	if patch.MaritalStatus != "" {
		out.MaritalStatus = patch.MaritalStatus
	}
	if patch.EmploymentType != "" {
		out.EmploymentType = patch.EmploymentType
	}
	if patch.EmployerName != "" {
		out.EmployerName = patch.EmployerName
	}
	if patch.YearsEmployed != 0 {
		out.YearsEmployed = patch.YearsEmployed
	}
	if patch.MonthlyNetIncome != 0 {
		out.MonthlyNetIncome = patch.MonthlyNetIncome
	}
	if patch.OtherMonthlyIncome != 0 {
		out.OtherMonthlyIncome = patch.OtherMonthlyIncome
	}
	if patch.MonthlyExpenses != 0 {
		out.MonthlyExpenses = patch.MonthlyExpenses
	}
	if patch.ExistingDebt != 0 {
		out.ExistingDebt = patch.ExistingDebt
	}
	if patch.AvailableDeposit != 0 {
		out.AvailableDeposit = patch.AvailableDeposit
	}
	if patch.DesiredPropertyPrice != 0 {
		out.DesiredPropertyPrice = patch.DesiredPropertyPrice
	}
	if patch.DesiredLoanAmount != 0 {
		out.DesiredLoanAmount = patch.DesiredLoanAmount
	}
	if patch.PreferredTenorYears != 0 {
		out.PreferredTenorYears = patch.PreferredTenorYears
	}
	if patch.SalaryMonths != 0 {
		out.SalaryMonths = patch.SalaryMonths
	}
	if patch.NHFContributorMonths != 0 {
		out.NHFContributorMonths = patch.NHFContributorMonths
	}
	if patch.WillingToDomicile {
		out.WillingToDomicile = true
	}
	return out
}

func (s *Service) syncProfiles(ctx context.Context, userID uuid.UUID, in AssessmentInput) error {
	if in.FullName != "" || in.CountryCode != "" {
		_, _ = s.db.ExecContext(ctx, `UPDATE user_profiles SET
			full_name = CASE WHEN $2 <> '' THEN $2 ELSE full_name END,
			state_of_residence = COALESCE(NULLIF($3,''), state_of_residence),
			marital_status = COALESCE(NULLIF($4,''), marital_status),
			residency_type = COALESCE(NULLIF($5,''), residency_type),
			country_code = COALESCE(NULLIF($6,''), country_code),
			updated_at = NOW()
			WHERE user_id = $1`,
			userID, in.FullName, in.StateOfResidence, in.MaritalStatus, in.ResidencyType, in.CountryCode)
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE employment_profiles SET
			employment_type = COALESCE(NULLIF($2,''), employment_type),
			employer_name = COALESCE(NULLIF($3,''), employer_name),
			years_employed = CASE WHEN $4 > 0 THEN $4 ELSE years_employed END,
			monthly_net_income = CASE WHEN $5 > 0 THEN $5 ELSE monthly_net_income END,
			other_monthly_income = CASE WHEN $6 > 0 THEN $6 ELSE other_monthly_income END,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, in.EmploymentType, in.EmployerName, in.YearsEmployed, in.MonthlyNetIncome, in.OtherMonthlyIncome)
	_, _ = s.db.ExecContext(ctx, `
		UPDATE financial_profiles SET
			monthly_expenses = CASE WHEN $2 > 0 THEN $2 ELSE monthly_expenses END,
			existing_debt_repayments = CASE WHEN $3 > 0 THEN $3 ELSE existing_debt_repayments END,
			available_deposit = CASE WHEN $4 > 0 THEN $4 ELSE available_deposit END,
			desired_property_price = CASE WHEN $5 > 0 THEN $5 ELSE desired_property_price END,
			desired_loan_amount = CASE WHEN $6 > 0 THEN $6 ELSE desired_loan_amount END,
			preferred_tenor_years = CASE WHEN $7 > 0 THEN $7 ELSE preferred_tenor_years END,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, in.MonthlyExpenses, in.ExistingDebt, in.AvailableDeposit, in.DesiredPropertyPrice, in.DesiredLoanAmount, in.PreferredTenorYears)
	return nil
}

func ageFromDOB(dob string) int {
	t, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return 0
	}
	now := time.Now()
	age := now.Year() - t.Year()
	if now.YearDay() < t.YearDay() {
		age--
	}
	return age
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
