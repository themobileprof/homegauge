package documents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/platform/storage"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidFile   = errors.New("invalid file")
	ErrApplicationNF = errors.New("application not found")
)

const maxUploadBytes = 10 << 20 // 10 MiB

var allowedMIME = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

type ChecklistItem struct {
	DocumentTypeCode string     `json:"document_type_code"`
	Label            string     `json:"label"`
	Category         string     `json:"category"`
	Required         bool       `json:"required"`
	Instructions     string     `json:"instructions"`
	Status           string     `json:"status"`
	DocumentID       *uuid.UUID `json:"document_id,omitempty"`
	UploadedAt       *time.Time `json:"uploaded_at,omitempty"`
}

type Document struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	ApplicationID    *uuid.UUID `json:"application_id,omitempty"`
	DocumentTypeCode string     `json:"document_type_code"`
	StorageKey       string     `json:"-"`
	MimeType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	Checksum         string     `json:"checksum,omitempty"`
	Version          int        `json:"version"`
	Status           string     `json:"status"`
	UploadedAt       time.Time  `json:"uploaded_at"`
}

type Service struct {
	db    *sql.DB
	store *storage.LocalStore
}

func NewService(db *sql.DB, store *storage.LocalStore) *Service {
	return &Service{db: db, store: store}
}

func (s *Service) EnsureApplication(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM mortgage_applications
		WHERE user_id = $1 AND status NOT IN ('CANCELLED', 'COMPLETED', 'REJECTED')
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO mortgage_applications (user_id, status, next_action_text)
		VALUES ($1, 'DOCUMENTS_PENDING', 'Upload your required documents, starting with 6-month salary statements.')
		RETURNING id
	`, userID).Scan(&id)
	return id, err
}

func (s *Service) Checklist(ctx context.Context, userID uuid.UUID, productID *uuid.UUID) ([]ChecklistItem, uuid.UUID, error) {
	appID, err := s.EnsureApplication(ctx, userID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	// Prefer preferred product on application, else explicit product, else first likely-eligible, else union of common salary docs
	var pid uuid.UUID
	if productID != nil {
		pid = *productID
	} else {
		_ = s.db.QueryRowContext(ctx, `SELECT preferred_product_id FROM mortgage_applications WHERE id=$1`, appID).Scan(&pid)
		if pid == uuid.Nil {
			_ = s.db.QueryRowContext(ctx, `
				SELECT r.product_id
				FROM eligibility_results r
				JOIN eligibility_assessments a ON a.id = r.assessment_id
				WHERE a.user_id = $1 AND r.outcome IN ('likely_eligible','potentially_eligible')
				ORDER BY a.completed_at DESC NULLS LAST
				LIMIT 1
			`, userID).Scan(&pid)
		}
	}

	type req struct {
		code, label, category, instructions string
		required                            bool
	}
	var reqs []req
	if pid != uuid.Nil {
		rows, err := s.db.QueryContext(ctx, `
			SELECT document_type_code, label, category, required, instructions
			FROM mortgage_product_documents WHERE product_id = $1
			ORDER BY category, label
		`, pid)
		if err != nil {
			return nil, uuid.Nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var r req
			if err := rows.Scan(&r.code, &r.label, &r.category, &r.required, &r.instructions); err != nil {
				return nil, uuid.Nil, err
			}
			reqs = append(reqs, r)
		}
	}
	if len(reqs) == 0 {
		reqs = []req{
			{"valid_id", "Valid government ID", "identity", "", true},
			{"payslips_3m", "3 months’ payslips", "income", "", true},
			{"employment_letter", "Employment letter", "income", "", true},
			{"salary_statements_6m", "6 months’ salary account statements", "banking", "Show clear monthly salary credits.", true},
		}
	}

	items := make([]ChecklistItem, 0, len(reqs))
	for _, r := range reqs {
		item := ChecklistItem{
			DocumentTypeCode: r.code,
			Label:            r.label,
			Category:         r.category,
			Required:         r.required,
			Instructions:     r.instructions,
			Status:           "not_started",
		}
		var (
			docID  uuid.UUID
			status string
			upAt   time.Time
		)
		err := s.db.QueryRowContext(ctx, `
			SELECT id, status, uploaded_at FROM documents
			WHERE user_id = $1 AND document_type_code = $2
			ORDER BY version DESC LIMIT 1
		`, userID, r.code).Scan(&docID, &status, &upAt)
		if err == nil {
			item.Status = status
			item.DocumentID = &docID
			item.UploadedAt = &upAt
		}
		items = append(items, item)
	}
	return items, appID, nil
}

func (s *Service) Upload(ctx context.Context, userID uuid.UUID, appID uuid.UUID, docType, mime string, r io.Reader) (*Document, error) {
	if !allowedMIME[mime] {
		return nil, fmt.Errorf("%w: only PDF, JPG, and PNG are allowed", ErrInvalidFile)
	}
	if !s.validDocType(ctx, docType) {
		return nil, fmt.Errorf("%w: unknown document type", ErrInvalidFile)
	}
	var owner uuid.UUID
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM mortgage_applications WHERE id=$1`, appID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrApplicationNF
	}
	if err != nil {
		return nil, err
	}
	if owner != userID {
		return nil, ErrForbidden
	}

	key, err := storage.NewObjectKey(userID.String(), docType)
	if err != nil {
		return nil, err
	}
	ext := extForMIME(mime)
	key = key + ext

	limited := io.LimitReader(r, maxUploadBytes+1)
	size, checksum, err := s.store.Put(ctx, key, limited)
	if err != nil {
		return nil, err
	}
	if size > maxUploadBytes {
		return nil, fmt.Errorf("%w: file exceeds 10MB limit", ErrInvalidFile)
	}

	var version int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM documents
		WHERE user_id=$1 AND document_type_code=$2
	`, userID, docType).Scan(&version)
	version++

	var id uuid.UUID
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO documents (user_id, application_id, document_type_code, storage_key, mime_type, size_bytes, checksum, version, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'uploaded')
		RETURNING id
	`, userID, appID, docType, key, mime, size, checksum, version).Scan(&id)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE mortgage_applications
		SET status = CASE WHEN status IN ('NEW','ASSESSMENT_COMPLETED') THEN 'DOCUMENTS_PENDING' ELSE status END,
		    next_action_text = 'Continue uploading required documents or request advisor review.',
		    updated_at = NOW()
		WHERE id = $1
	`, appID)

	return s.Get(ctx, userID, id)
}

func (s *Service) Get(ctx context.Context, userID, docID uuid.UUID) (*Document, error) {
	d, err := s.scanDocument(ctx, docID)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, ErrForbidden
	}
	return d, nil
}

func (s *Service) GetForStaff(ctx context.Context, docID uuid.UUID) (*Document, error) {
	return s.scanDocument(ctx, docID)
}

func (s *Service) scanDocument(ctx context.Context, docID uuid.UUID) (*Document, error) {
	var d Document
	var appID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, application_id::text, document_type_code, storage_key, mime_type, size_bytes, COALESCE(checksum,''), version, status, uploaded_at
		FROM documents WHERE id=$1
	`, docID).Scan(&d.ID, &d.UserID, &appID, &d.DocumentTypeCode, &d.StorageKey, &d.MimeType, &d.SizeBytes, &d.Checksum, &d.Version, &d.Status, &d.UploadedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if appID.Valid {
		id, err := uuid.Parse(appID.String)
		if err == nil {
			d.ApplicationID = &id
		}
	}
	return &d, nil
}

func (s *Service) DownloadURL(ctx context.Context, userID, docID uuid.UUID, staff bool) (string, time.Time, error) {
	var d *Document
	var err error
	if staff {
		d, err = s.GetForStaff(ctx, docID)
	} else {
		d, err = s.Get(ctx, userID, docID)
	}
	if err != nil {
		return "", time.Time{}, err
	}
	token, exp := s.store.Sign(d.StorageKey, 5*time.Minute)
	return token, exp, nil
}

func (s *Service) OpenByToken(ctx context.Context, token string) (io.ReadCloser, string, string, error) {
	key, err := s.store.Verify(token)
	if err != nil {
		return nil, "", "", err
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, "", "", err
	}
	var mime string
	_ = s.db.QueryRowContext(ctx, `SELECT mime_type FROM documents WHERE storage_key=$1 ORDER BY uploaded_at DESC LIMIT 1`, key).Scan(&mime)
	if mime == "" {
		mime = "application/octet-stream"
	}
	name := filepath.Base(key)
	return f, mime, name, nil
}

func (s *Service) Review(ctx context.Context, reviewerID, docID uuid.UUID, decision, notes string) error {
	decision = strings.ToLower(decision)
	status := map[string]string{
		"accepted":             "accepted",
		"rejected":             "rejected",
		"requires_replacement": "requires_replacement",
		"under_review":         "under_review",
	}[decision]
	if status == "" {
		return ErrInvalidFile
	}
	res, err := s.db.ExecContext(ctx, `UPDATE documents SET status=$2, updated_at=NOW() WHERE id=$1`, docID, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO document_reviews (document_id, reviewer_id, decision, notes)
		VALUES ($1,$2,$3,$4)
	`, docID, reviewerID, decision, notes)
	return err
}

func (s *Service) validDocType(ctx context.Context, code string) bool {
	var exists bool
	_ = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM document_types WHERE code=$1)`, code).Scan(&exists)
	return exists
}

func extForMIME(mime string) string {
	switch mime {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}
