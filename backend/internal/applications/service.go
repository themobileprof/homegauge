package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid status")
)

type Application struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	CustomerEmail     string     `json:"customer_email,omitempty"`
	CustomerName      string     `json:"customer_name,omitempty"`
	AssessmentID      *uuid.UUID `json:"assessment_id,omitempty"`
	PreferredProductID *uuid.UUID `json:"preferred_product_id,omitempty"`
	Status            string     `json:"status"`
	AssignedAdvisorID *uuid.UUID `json:"assigned_advisor_id,omitempty"`
	NextActionText    string     `json:"next_action_text"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Note struct {
	ID         uuid.UUID `json:"id"`
	AuthorID   uuid.UUID `json:"author_id"`
	AuthorEmail string   `json:"author_email,omitempty"`
	Body       string    `json:"body"`
	Visibility string    `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
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
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) GetMine(ctx context.Context, userID uuid.UUID) (*Application, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, assessment_id::text, preferred_product_id::text, status, assigned_advisor_id::text, next_action_text, created_at, updated_at
		FROM mortgage_applications
		WHERE user_id=$1 AND status NOT IN ('CANCELLED')
		ORDER BY created_at DESC LIMIT 1
	`, userID)
	return scanApp(row, "", "")
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
	// Create a concierge suggestion for human review
	payload := []byte(`{"message":"Customer requested human assistance. Review eligibility results and document checklist.","actions":["review_documents","message_customer"]}`)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO concierge_suggestions (application_id, suggestion_type, payload, confidence, status)
		VALUES ($1, 'request_human_review', $2::jsonb, 0.9, 'pending')
	`, app.ID, string(payload))
	return s.GetByID(ctx, app.ID)
}

func (s *Service) ListCases(ctx context.Context) ([]Application, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.user_id, a.assessment_id::text, a.preferred_product_id::text, a.status,
			a.assigned_advisor_id::text, a.next_action_text, a.created_at, a.updated_at,
			u.email, COALESCE(p.full_name,'')
		FROM mortgage_applications a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN user_profiles p ON p.user_id = a.user_id
		WHERE a.status NOT IN ('CANCELLED')
		ORDER BY a.updated_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Application
	for rows.Next() {
		var (
			a Application
			assessment, preferred, advisor sql.NullString
			email, name string
		)
		if err := rows.Scan(&a.ID, &a.UserID, &assessment, &preferred, &a.Status, &advisor, &a.NextActionText, &a.CreatedAt, &a.UpdatedAt, &email, &name); err != nil {
			return nil, err
		}
		a.CustomerEmail = email
		a.CustomerName = name
		a.AssessmentID = parseNullUUID(assessment)
		a.PreferredProductID = parseNullUUID(preferred)
		a.AssignedAdvisorID = parseNullUUID(advisor)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.user_id, a.assessment_id::text, a.preferred_product_id::text, a.status,
			a.assigned_advisor_id::text, a.next_action_text, a.created_at, a.updated_at,
			u.email, COALESCE(p.full_name,'')
		FROM mortgage_applications a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN user_profiles p ON p.user_id = a.user_id
		WHERE a.id=$1
	`, id)
	var (
		a Application
		assessment, preferred, advisor sql.NullString
		email, name string
	)
	err := row.Scan(&a.ID, &a.UserID, &assessment, &preferred, &a.Status, &advisor, &a.NextActionText, &a.CreatedAt, &a.UpdatedAt, &email, &name)
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

func (s *Service) UpdateStatus(ctx context.Context, actorID, appID uuid.UUID, status, nextAction string) (*Application, error) {
	allowed := map[string]bool{
		"NEW": true, "ASSESSMENT_COMPLETED": true, "DOCUMENTS_PENDING": true, "DOCUMENTS_UNDER_REVIEW": true,
		"READY_FOR_SUBMISSION": true, "SUBMITTED_TO_LENDER": true, "LENDER_REVIEW": true,
		"ADDITIONAL_INFORMATION_REQUIRED": true, "APPROVED": true, "REJECTED": true, "COMPLETED": true, "CANCELLED": true,
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

func (s *Service) Assign(ctx context.Context, actorID, appID, advisorID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE mortgage_applications SET assigned_advisor_id=$2, updated_at=NOW() WHERE id=$1
	`, appID, advisorID)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE advisor_assignments SET active=FALSE WHERE application_id=$1 AND active=TRUE`, appID)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO advisor_assignments (application_id, advisor_id, assigned_by, active)
		VALUES ($1,$2,$3,TRUE)
	`, appID, advisorID, actorID)
	return err
}

func (s *Service) AddNote(ctx context.Context, authorID, appID uuid.UUID, body, visibility string) (*Note, error) {
	if visibility == "" {
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
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.author_id, u.email, n.body, n.visibility, n.created_at
		FROM advisor_notes n JOIN users u ON u.id = n.author_id
		WHERE n.application_id=$1 ORDER BY n.created_at DESC
	`, appID)
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
