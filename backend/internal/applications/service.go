package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalid        = errors.New("invalid status")
	ErrInvalidAdvisor = errors.New("invalid advisor")
)

type Application struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	CustomerEmail      string     `json:"customer_email,omitempty"`
	CustomerName       string     `json:"customer_name,omitempty"`
	AssessmentID       *uuid.UUID `json:"assessment_id,omitempty"`
	PreferredProductID   *uuid.UUID `json:"preferred_product_id,omitempty"`
	PreferredProductName string     `json:"preferred_product_name,omitempty"`
	LenderID             *uuid.UUID `json:"lender_id,omitempty"`
	LenderName           string     `json:"lender_name,omitempty"`
	LenderHasAccount     bool       `json:"lender_has_account,omitempty"`
	Status               string     `json:"status"`
	AssignedAdvisorID    *uuid.UUID `json:"assigned_advisor_id,omitempty"`
	AssignedAdvisorName  string     `json:"assigned_advisor_name,omitempty"`
	AssignedAdvisorEmail string     `json:"assigned_advisor_email,omitempty"`
	NextActionText       string     `json:"next_action_text"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Note struct {
	ID          uuid.UUID `json:"id"`
	AuthorID    uuid.UUID `json:"author_id"`
	AuthorEmail string    `json:"author_email,omitempty"`
	Body        string    `json:"body"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
}

type Suggestion struct {
	ID             uuid.UUID       `json:"id"`
	SuggestionType string          `json:"suggestion_type"`
	Payload        json.RawMessage `json:"payload"`
	Confidence     *float64        `json:"confidence,omitempty"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Service struct {
	db      *sql.DB
	funding FundingSyncer
}

// FundingSyncer creates/updates pre-disbursement obligations when a product is chosen.
type FundingSyncer interface {
	SyncObligations(ctx context.Context, appID, productID uuid.UUID) error
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) SetFundingSync(f FundingSyncer) { s.funding = f }

func (s *Service) GetMine(ctx context.Context, userID uuid.UUID) (*Application, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM mortgage_applications
		WHERE user_id=$1 AND status NOT IN ('CANCELLED')
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) RequestAdvisor(ctx context.Context, userID uuid.UUID) (*Application, error) {
	app, err := s.GetMine(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		var id uuid.UUID
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO mortgage_applications (user_id, status, next_action_text)
			VALUES ($1, 'DOCUMENTS_PENDING', 'An advisor will review your case shortly.')
			RETURNING id
		`, userID).Scan(&id)
		if err != nil {
			return nil, err
		}
		app, err = s.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE mortgage_applications
		SET next_action_text = 'Advisor assistance requested. We will review your documents and eligibility.',
		    updated_at = NOW()
		WHERE id = $1
	`, app.ID)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO application_events (application_id, actor_id, event_type, payload)
		VALUES ($1, $2, 'advisor_requested', '{"source":"customer"}')
	`, app.ID, userID)

	_ = s.createProgrammaticSuggestions(ctx, app)
	return s.GetByID(ctx, app.ID)
}

// createProgrammaticSuggestions builds advisor next steps from DB facts — no LLM.
func (s *Service) createProgrammaticSuggestions(ctx context.Context, app *Application) error {
	facts, err := s.loadCaseFacts(ctx, app)
	if err != nil {
		return err
	}
	draft := buildAdvisorDraft(facts)
	payload, _ := json.Marshal(draft)
	conf := 0.95
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO concierge_suggestions (application_id, suggestion_type, payload, confidence, status)
		VALUES ($1, 'advisor_checklist', $2::jsonb, $3, 'pending')
	`, app.ID, string(payload), conf)
	return err
}

type caseFacts struct {
	CustomerName   string
	Readiness      *int
	Likely         int
	Potential      int
	Unlikely       int
	MissingDocs    []string
	UploadedDocs   []string
	RejectedDocs   []string
	HasAssessment  bool
	SalaryMonths   int
	EquityPct      *float64
}

type advisorDraft struct {
	Message   string   `json:"message"`
	Actions   []string `json:"actions"`
	Priority  string   `json:"priority"`
	Rationale string   `json:"rationale"`
	Source    string   `json:"source"`
}

func (s *Service) loadCaseFacts(ctx context.Context, app *Application) (caseFacts, error) {
	f := caseFacts{CustomerName: app.CustomerName}
	if f.CustomerName == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(full_name,'') FROM user_profiles WHERE user_id=$1`, app.UserID).Scan(&f.CustomerName)
	}

	if app.AssessmentID != nil {
		f.HasAssessment = true
		var score sql.NullInt64
		_ = s.db.QueryRowContext(ctx, `
			SELECT total_score FROM readiness_scores WHERE assessment_id=$1 ORDER BY created_at DESC LIMIT 1
		`, *app.AssessmentID).Scan(&score)
		if score.Valid {
			v := int(score.Int64)
			f.Readiness = &v
		}
		rows, err := s.db.QueryContext(ctx, `
			SELECT outcome FROM eligibility_results WHERE assessment_id=$1
		`, *app.AssessmentID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var o string
				if rows.Scan(&o) == nil {
					switch o {
					case "likely_eligible":
						f.Likely++
					case "potentially_eligible", "may_require_review":
						f.Potential++
					case "unlikely", "more_info_required":
						f.Unlikely++
					}
				}
			}
		}
		var snap []byte
		_ = s.db.QueryRowContext(ctx, `SELECT input_snapshot FROM eligibility_assessments WHERE id=$1`, *app.AssessmentID).Scan(&snap)
		if len(snap) > 0 {
			var in struct {
				SalaryMonths         int     `json:"salary_months"`
				AvailableDeposit     float64 `json:"available_deposit"`
				DesiredPropertyPrice float64 `json:"desired_property_price"`
			}
			if json.Unmarshal(snap, &in) == nil {
				f.SalaryMonths = in.SalaryMonths
				if in.DesiredPropertyPrice > 0 {
					pct := (in.AvailableDeposit / in.DesiredPropertyPrice) * 100
					f.EquityPct = &pct
				}
			}
		}
	}

	// Required docs for linked / any active product checklist style: types on application uploads + common required set.
	uploaded := map[string]string{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT document_type_code, status FROM documents WHERE application_id=$1
	`, app.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code, status string
			if rows.Scan(&code, &status) == nil {
				uploaded[code] = status
				if status == "rejected" || status == "requires_replacement" {
					f.RejectedDocs = append(f.RejectedDocs, code)
				} else if status == "uploaded" || status == "under_review" || status == "accepted" {
					f.UploadedDocs = append(f.UploadedDocs, code)
				}
			}
		}
	}

	required := []string{"salary_statements_6m", "valid_id", "payslips_3m", "employment_letter"}
	// Prefer product-linked requirements when assessment exists
	if app.AssessmentID != nil {
		prows, perr := s.db.QueryContext(ctx, `
			SELECT DISTINCT mpd.document_type_code
			FROM eligibility_results er
			JOIN mortgage_product_documents mpd ON mpd.product_id = er.product_id
			WHERE er.assessment_id = $1 AND mpd.required = TRUE
		`, *app.AssessmentID)
		if perr == nil {
			defer prows.Close()
			var codes []string
			for prows.Next() {
				var code string
				if prows.Scan(&code) == nil {
					codes = append(codes, code)
				}
			}
			if len(codes) > 0 {
				required = codes
			}
		}
	}
	for _, code := range required {
		st, ok := uploaded[code]
		if !ok || st == "not_started" || st == "rejected" || st == "requires_replacement" {
			f.MissingDocs = append(f.MissingDocs, code)
		}
	}
	return f, nil
}

func buildAdvisorDraft(f caseFacts) advisorDraft {
	actions := []string{"message_customer"}
	var parts []string
	priority := "medium"

	if !f.HasAssessment {
		parts = append(parts, "No completed eligibility assessment yet — ask the customer to finish Check Eligibility.")
		actions = append(actions, "request_assessment")
		priority = "high"
	} else {
		if f.Readiness != nil {
			parts = append(parts, fmt.Sprintf("Readiness score is %d/100.", *f.Readiness))
			if *f.Readiness < 55 {
				priority = "high"
			}
		}
		parts = append(parts, fmt.Sprintf("Product fit: %d likely, %d needs review, %d unlikely.", f.Likely, f.Potential, f.Unlikely))
		if f.Likely == 0 && f.Potential == 0 {
			priority = "high"
			actions = append(actions, "review_eligibility_inputs")
			parts = append(parts, "No strong product matches — verify income, deposit, and tenor before escalating.")
		} else {
			actions = append(actions, "confirm_best_fit_product")
		}
		if f.SalaryMonths > 0 && f.SalaryMonths < 6 {
			priority = "high"
			parts = append(parts, fmt.Sprintf("Only %d salary months declared (target ~6).", f.SalaryMonths))
			actions = append(actions, "verify_salary_pattern")
		}
		if f.EquityPct != nil && *f.EquityPct < 10 {
			parts = append(parts, fmt.Sprintf("Declared equity is about %.0f%% — many products need ~10%%+.", *f.EquityPct))
			actions = append(actions, "discuss_deposit_gap")
		}
	}

	if len(f.MissingDocs) > 0 {
		if priority != "high" {
			priority = "medium"
		}
		parts = append(parts, "Missing or incomplete documents: "+strings.Join(humanizeDocCodes(f.MissingDocs), ", ")+".")
		actions = append(actions, "chase_missing_documents")
		for _, code := range f.MissingDocs {
			if code == "salary_statements_6m" {
				actions = append(actions, "collect_salary_statements")
				break
			}
		}
	} else if len(f.UploadedDocs) > 0 {
		parts = append(parts, "Core documents are on file — review uploads and mark accept/reject.")
		actions = append(actions, "review_uploaded_documents")
	} else {
		parts = append(parts, "No documents uploaded yet.")
		actions = append(actions, "chase_missing_documents")
		priority = "high"
	}
	if len(f.RejectedDocs) > 0 {
		priority = "high"
		parts = append(parts, "Some documents need replacement: "+strings.Join(humanizeDocCodes(f.RejectedDocs), ", ")+".")
		actions = append(actions, "request_document_replacements")
	}

	name := f.CustomerName
	if name == "" {
		name = "Customer"
	}
	msg := fmt.Sprintf("%s requested advisor help. %s", name, strings.Join(parts, " "))
	actions = uniqueStrings(actions)

	return advisorDraft{
		Message:   msg,
		Actions:   actions,
		Priority:  priority,
		Rationale: "Generated from assessment outcomes, readiness score, and document checklist — not an LLM.",
		Source:    "rules",
	}
}

func humanizeDocCodes(codes []string) []string {
	labels := map[string]string{
		"salary_statements_6m": "6-month salary statements",
		"valid_id":             "valid ID",
		"payslips_3m":          "3 months’ payslips",
		"employment_letter":    "employment letter",
		"nhf_evidence":         "NHF evidence",
		"offer_letter":         "property offer letter",
		"title_docs":           "title documents",
		"passport_photo":       "passport photo",
	}
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if l, ok := labels[c]; ok {
			out = append(out, l)
		} else {
			out = append(out, strings.ReplaceAll(c, "_", " "))
		}
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

type CaseQuery struct {
	AssignedTo    *uuid.UUID
	Unassigned    bool
	Status        string
	Statuses      []string
	IncludeClosed bool
	LenderID      *uuid.UUID
	Limit         int
}

func (s *Service) ListAdvisorCases(ctx context.Context, advisorID uuid.UUID) ([]Application, error) {
	return s.ListCases(ctx, CaseQuery{AssignedTo: &advisorID, Limit: 100})
}

func (s *Service) ListCases(ctx context.Context, q CaseQuery) ([]Application, error) {
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	sqlStr := `
		SELECT a.id, a.user_id, a.assessment_id::text, a.preferred_product_id::text, a.status,
			a.assigned_advisor_id::text, a.next_action_text, a.created_at, a.updated_at,
			u.email, COALESCE(p.full_name,''),
			COALESCE(advp.full_name,''), COALESCE(advu.email,''),
			COALESCE(mp.name,''), COALESCE(ln.name,''), ln.id::text,
			EXISTS(
				SELECT 1 FROM users lu
				WHERE lu.lender_id = ln.id AND lu.role = 'LENDER_USER'
				  AND lu.status = 'active' AND lu.deleted_at IS NULL
			)
		FROM mortgage_applications a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN user_profiles p ON p.user_id = a.user_id
		LEFT JOIN users advu ON advu.id = a.assigned_advisor_id
		LEFT JOIN user_profiles advp ON advp.user_id = a.assigned_advisor_id
		LEFT JOIN mortgage_products mp ON mp.id = a.preferred_product_id AND mp.deleted_at IS NULL
		LEFT JOIN lenders ln ON ln.id = mp.lender_id AND ln.deleted_at IS NULL
		WHERE 1=1`
	args := []any{}
	n := 1
	if q.AssignedTo != nil {
		sqlStr += fmt.Sprintf(` AND a.assigned_advisor_id = $%d`, n)
		args = append(args, *q.AssignedTo)
		n++
	}
	if q.LenderID != nil {
		sqlStr += fmt.Sprintf(` AND mp.lender_id = $%d`, n)
		args = append(args, *q.LenderID)
		n++
	}
	if q.Unassigned {
		sqlStr += ` AND a.assigned_advisor_id IS NULL`
	}
	if q.Status != "" {
		sqlStr += fmt.Sprintf(` AND a.status = $%d`, n)
		args = append(args, q.Status)
		n++
	}
	if len(q.Statuses) > 0 {
		sqlStr += ` AND a.status IN (`
		for i, st := range q.Statuses {
			if i > 0 {
				sqlStr += `,`
			}
			sqlStr += fmt.Sprintf(`$%d`, n)
			args = append(args, st)
			n++
		}
		sqlStr += `)`
	}
	if !q.IncludeClosed && q.Status == "" && len(q.Statuses) == 0 {
		sqlStr += ` AND a.status NOT IN ('CANCELLED')`
	}
	sqlStr += fmt.Sprintf(` ORDER BY a.updated_at DESC LIMIT $%d`, n)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Application{}
	for rows.Next() {
		a, err := scanListedApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.user_id, a.assessment_id::text, a.preferred_product_id::text, a.status,
			a.assigned_advisor_id::text, a.next_action_text, a.created_at, a.updated_at,
			u.email, COALESCE(p.full_name,''),
			COALESCE(advp.full_name,''), COALESCE(advu.email,''),
			COALESCE(mp.name,''), COALESCE(ln.name,''), ln.id::text,
			EXISTS(
				SELECT 1 FROM users lu
				WHERE lu.lender_id = ln.id AND lu.role = 'LENDER_USER'
				  AND lu.status = 'active' AND lu.deleted_at IS NULL
			)
		FROM mortgage_applications a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN user_profiles p ON p.user_id = a.user_id
		LEFT JOIN users advu ON advu.id = a.assigned_advisor_id
		LEFT JOIN user_profiles advp ON advp.user_id = a.assigned_advisor_id
		LEFT JOIN mortgage_products mp ON mp.id = a.preferred_product_id AND mp.deleted_at IS NULL
		LEFT JOIN lenders ln ON ln.id = mp.lender_id AND ln.deleted_at IS NULL
		WHERE a.id=$1
	`, id)
	a, err := scanListedApp(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) GetAdvisorCase(ctx context.Context, advisorID, appID uuid.UUID) (*Application, error) {
	app, err := s.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.AssignedAdvisorID == nil || *app.AssignedAdvisorID != advisorID {
		return nil, ErrForbidden
	}
	return app, nil
}

func (s *Service) SetPreferredProduct(ctx context.Context, actorID, appID, productID uuid.UUID) (*Application, error) {
	var name, status string
	err := s.db.QueryRowContext(ctx, `
		SELECT name, status FROM mortgage_products WHERE id=$1 AND deleted_at IS NULL
	`, productID).Scan(&name, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, ErrInvalid
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE mortgage_applications SET preferred_product_id=$2, updated_at=NOW() WHERE id=$1
	`, appID, productID)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO application_events (application_id, actor_id, event_type, payload)
		VALUES ($1,$2,'preferred_product_set', jsonb_build_object('product_id', $3::text, 'product_name', $4::text))
	`, appID, actorID, productID, name)
	if s.funding != nil {
		_ = s.funding.SyncObligations(ctx, appID, productID)
	}
	return s.GetByID(ctx, appID)
}

func (s *Service) GetLenderCase(ctx context.Context, lenderID, appID uuid.UUID) (*Application, error) {
	app, err := s.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.LenderID == nil || *app.LenderID != lenderID {
		return nil, ErrForbidden
	}
	if !lenderVisibleStatus(app.Status) {
		return nil, ErrForbidden
	}
	return app, nil
}

func (s *Service) ListLenderCases(ctx context.Context, lenderID uuid.UUID) ([]Application, error) {
	return s.ListCases(ctx, CaseQuery{
		LenderID:      &lenderID,
		Statuses:      []string{"SUBMITTED_TO_LENDER", "LENDER_REVIEW", "ADDITIONAL_INFORMATION_REQUIRED", "APPROVED", "REJECTED", "COMPLETED"},
		IncludeClosed: true,
		Limit:         100,
	})
}

func lenderVisibleStatus(status string) bool {
	switch status {
	case "SUBMITTED_TO_LENDER", "LENDER_REVIEW", "ADDITIONAL_INFORMATION_REQUIRED", "APPROVED", "REJECTED", "COMPLETED":
		return true
	default:
		return false
	}
}

type LenderOrg struct {
	ID          uuid.UUID `json:"id"`
	CountryCode string    `json:"country_code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

func (s *Service) LenderOrg(ctx context.Context, lenderID uuid.UUID) (*LenderOrg, error) {
	var o LenderOrg
	err := s.db.QueryRowContext(ctx, `
		SELECT id, country_code, name, description FROM lenders
		WHERE id=$1 AND deleted_at IS NULL
	`, lenderID).Scan(&o.ID, &o.CountryCode, &o.Name, &o.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Service) UpdateStatus(ctx context.Context, actorID, appID uuid.UUID, status, nextAction string, allowed map[string]bool) (*Application, error) {
	if allowed == nil {
		allowed = allCaseStatuses
	}
	if !allowed[status] {
		return nil, ErrInvalid
	}
	var from string
	_ = s.db.QueryRowContext(ctx, `SELECT status FROM mortgage_applications WHERE id=$1`, appID).Scan(&from)
	_, err := s.db.ExecContext(ctx, `
		UPDATE mortgage_applications SET status=$2, next_action_text=COALESCE(NULLIF($3,''), next_action_text), updated_at=NOW()
		WHERE id=$1
	`, appID, status, nextAction)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO application_events (application_id, actor_id, event_type, from_status, to_status, payload)
		VALUES ($1,$2,'status_changed',$3,$4,'{}')
	`, appID, actorID, from, status)
	return s.GetByID(ctx, appID)
}

func (s *Service) Assign(ctx context.Context, actorID, appID, advisorID uuid.UUID) (*Application, error) {
	if _, err := s.GetByID(ctx, appID); err != nil {
		return nil, err
	}
	var role, ustatus, email, name string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.role::text, u.status, u.email, COALESCE(p.full_name,'')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id=$1 AND u.deleted_at IS NULL
	`, advisorID).Scan(&role, &ustatus, &email, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidAdvisor
	}
	if err != nil {
		return nil, err
	}
	if role != "ADVISOR" || ustatus != "active" {
		return nil, ErrInvalidAdvisor
	}
	label := name
	if label == "" {
		label = email
	}
	next := fmt.Sprintf("Assigned to %s.", label)
	_, err = s.db.ExecContext(ctx, `
		UPDATE mortgage_applications SET assigned_advisor_id=$2, next_action_text=$3, updated_at=NOW() WHERE id=$1
	`, appID, advisorID, next)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE advisor_assignments SET active=FALSE WHERE application_id=$1 AND active=TRUE`, appID)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO advisor_assignments (application_id, advisor_id, assigned_by, active)
		VALUES ($1,$2,$3,TRUE)
	`, appID, advisorID, actorID)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO application_events (application_id, actor_id, event_type, payload)
		VALUES ($1,$2,'assigned', jsonb_build_object('advisor_id', $3::text))
	`, appID, actorID, advisorID)
	return s.GetByID(ctx, appID)
}

func (s *Service) AddNote(ctx context.Context, authorID, appID uuid.UUID, body, visibility string) (*Note, error) {
	if visibility == "" {
		visibility = "internal"
	}
	switch visibility {
	case "internal", "customer", "lender":
	default:
		visibility = "internal"
	}
	var id uuid.UUID
	var created time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO advisor_notes (application_id, author_id, body, visibility)
		VALUES ($1,$2,$3,$4) RETURNING id, created_at
	`, appID, authorID, body, visibility).Scan(&id, &created)
	if err != nil {
		return nil, err
	}
	return &Note{ID: id, AuthorID: authorID, Body: body, Visibility: visibility, CreatedAt: created}, nil
}

func (s *Service) ListNotes(ctx context.Context, appID uuid.UUID) ([]Note, error) {
	return s.ListNotesVisible(ctx, appID, nil)
}

func (s *Service) ListNotesVisible(ctx context.Context, appID uuid.UUID, vis []string) ([]Note, error) {
	q := `
		SELECT n.id, n.author_id, u.email, n.body, n.visibility, n.created_at
		FROM advisor_notes n JOIN users u ON u.id = n.author_id
		WHERE n.application_id=$1`
	args := []any{appID}
	if len(vis) > 0 {
		q += ` AND n.visibility IN (`
		for i, v := range vis {
			if i > 0 {
				q += `,`
			}
			q += fmt.Sprintf(`$%d`, i+2)
			args = append(args, v)
		}
		q += `)`
	}
	q += ` ORDER BY n.created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.AuthorID, &n.AuthorEmail, &n.Body, &n.Visibility, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Service) ListSuggestions(ctx context.Context, appID uuid.UUID) ([]Suggestion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, suggestion_type, payload, confidence, status, created_at
		FROM concierge_suggestions WHERE application_id=$1 ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Suggestion
	for rows.Next() {
		var sg Suggestion
		var conf sql.NullFloat64
		if err := rows.Scan(&sg.ID, &sg.SuggestionType, &sg.Payload, &conf, &sg.Status, &sg.CreatedAt); err != nil {
			return nil, err
		}
		if conf.Valid {
			v := conf.Float64
			sg.Confidence = &v
		}
		out = append(out, sg)
	}
	return out, rows.Err()
}

func (s *Service) ResolveSuggestion(ctx context.Context, reviewerID, suggestionID uuid.UUID, status string) error {
	if status != "approved" && status != "rejected" && status != "applied" {
		return ErrInvalid
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE concierge_suggestions SET status=$2, reviewed_by=$3, reviewed_at=NOW() WHERE id=$1
	`, suggestionID, status, reviewerID)
	return err
}

func scanApp(row *sql.Row, email, name string) (*Application, error) {
	var (
		a Application
		assessment, preferred, advisor sql.NullString
	)
	err := row.Scan(&a.ID, &a.UserID, &assessment, &preferred, &a.Status, &advisor, &a.NextActionText, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CustomerEmail = email
	a.CustomerName = name
	a.AssessmentID = parseNullUUID(assessment)
	a.PreferredProductID = parseNullUUID(preferred)
	a.AssignedAdvisorID = parseNullUUID(advisor)
	return &a, nil
}

func scanListedApp(row interface{ Scan(dest ...any) error }) (Application, error) {
	var a Application
	var assessment, preferred, advisor, lenderID sql.NullString
	var email, name, advName, advEmail, productName, lenderName string
	err := row.Scan(
		&a.ID, &a.UserID, &assessment, &preferred, &a.Status, &advisor, &a.NextActionText, &a.CreatedAt, &a.UpdatedAt,
		&email, &name, &advName, &advEmail, &productName, &lenderName, &lenderID, &a.LenderHasAccount,
	)
	if err != nil {
		return a, err
	}
	a.CustomerEmail = email
	a.CustomerName = name
	a.AssignedAdvisorName = advName
	a.AssignedAdvisorEmail = advEmail
	a.PreferredProductName = productName
	a.LenderName = lenderName
	a.AssessmentID = parseNullUUID(assessment)
	a.PreferredProductID = parseNullUUID(preferred)
	a.AssignedAdvisorID = parseNullUUID(advisor)
	a.LenderID = parseNullUUID(lenderID)
	return a, nil
}

func parseNullUUID(ns sql.NullString) *uuid.UUID {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	id, err := uuid.Parse(ns.String)
	if err != nil {
		return nil
	}
	return &id
}
