package applications

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type AdvisorStaff struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	OpenCases int    `json:"open_cases"`
}

type AdvisorReportRow struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	FullName       string `json:"full_name"`
	OpenCases      int    `json:"open_cases"`
	AssignedTotal  int    `json:"assigned_total"`
	Completed      int    `json:"completed"`
	Approved       int    `json:"approved"`
	Rejected       int    `json:"rejected"`
}

type BuyerReportRow struct {
	UserID               string     `json:"user_id"`
	Email                string     `json:"email"`
	FullName             string     `json:"full_name"`
	CaseID               *uuid.UUID `json:"case_id,omitempty"`
	Status               string     `json:"status,omitempty"`
	AssignedAdvisorName  string     `json:"assigned_advisor_name,omitempty"`
	AssignedAdvisorEmail string     `json:"assigned_advisor_email,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

type ReportSummary struct {
	UnassignedOpen int            `json:"unassigned_open"`
	OpenCases      int            `json:"open_cases"`
	ByStatus       map[string]int `json:"by_status"`
}

func (s *Service) ListActiveAdvisors(ctx context.Context) ([]AdvisorStaff, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id::text, u.email, COALESCE(p.full_name,''),
			(SELECT COUNT(*) FROM mortgage_applications a
			 WHERE a.assigned_advisor_id = u.id AND a.status NOT IN ('CANCELLED','COMPLETED','REJECTED'))
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.role = 'ADVISOR' AND u.status = 'active' AND u.deleted_at IS NULL
		ORDER BY COALESCE(NULLIF(p.full_name,''), u.email)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdvisorStaff{}
	for rows.Next() {
		var it AdvisorStaff
		if err := rows.Scan(&it.ID, &it.Email, &it.FullName, &it.OpenCases); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) ReportAdvisors(ctx context.Context) ([]AdvisorReportRow, ReportSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id::text, u.email, COALESCE(p.full_name,''),
			COUNT(*) FILTER (WHERE a.id IS NOT NULL AND a.status NOT IN ('CANCELLED','COMPLETED','REJECTED')),
			COUNT(*) FILTER (WHERE a.id IS NOT NULL),
			COUNT(*) FILTER (WHERE a.status = 'COMPLETED'),
			COUNT(*) FILTER (WHERE a.status = 'APPROVED'),
			COUNT(*) FILTER (WHERE a.status = 'REJECTED')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN mortgage_applications a ON a.assigned_advisor_id = u.id
		WHERE u.role = 'ADVISOR' AND u.deleted_at IS NULL
		GROUP BY u.id, u.email, p.full_name
		ORDER BY COALESCE(NULLIF(p.full_name,''), u.email)
	`)
	if err != nil {
		return nil, ReportSummary{}, err
	}
	defer rows.Close()
	out := []AdvisorReportRow{}
	for rows.Next() {
		var it AdvisorReportRow
		if err := rows.Scan(&it.ID, &it.Email, &it.FullName, &it.OpenCases, &it.AssignedTotal, &it.Completed, &it.Approved, &it.Rejected); err != nil {
			return nil, ReportSummary{}, err
		}
		out = append(out, it)
	}
	sum, err := s.reportSummary(ctx)
	if err != nil {
		return nil, ReportSummary{}, err
	}
	return out, sum, rows.Err()
}

func (s *Service) ReportBuyers(ctx context.Context) ([]BuyerReportRow, ReportSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id::text, u.email, COALESCE(p.full_name,''),
			a.id, a.status, a.updated_at,
			COALESCE(advp.full_name,''), COALESCE(advu.email,'')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT id, status, updated_at, assigned_advisor_id
			FROM mortgage_applications
			WHERE user_id = u.id AND status <> 'CANCELLED'
			ORDER BY updated_at DESC
			LIMIT 1
		) a ON TRUE
		LEFT JOIN users advu ON advu.id = a.assigned_advisor_id
		LEFT JOIN user_profiles advp ON advp.user_id = a.assigned_advisor_id
		WHERE u.role = 'CUSTOMER' AND u.deleted_at IS NULL
		ORDER BY a.updated_at DESC NULLS LAST, u.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		return nil, ReportSummary{}, err
	}
	defer rows.Close()
	out := []BuyerReportRow{}
	for rows.Next() {
		var it BuyerReportRow
		var caseID uuid.NullUUID
		var status sql.NullString
		var updated sql.NullTime
		var advName, advEmail string
		if err := rows.Scan(&it.UserID, &it.Email, &it.FullName, &caseID, &status, &updated, &advName, &advEmail); err != nil {
			return nil, ReportSummary{}, err
		}
		if caseID.Valid {
			id := caseID.UUID
			it.CaseID = &id
		}
		if status.Valid {
			it.Status = status.String
		}
		if updated.Valid {
			t := updated.Time
			it.UpdatedAt = &t
		}
		it.AssignedAdvisorName = advName
		it.AssignedAdvisorEmail = advEmail
		out = append(out, it)
	}
	sum, err := s.reportSummary(ctx)
	if err != nil {
		return nil, ReportSummary{}, err
	}
	return out, sum, rows.Err()
}

func (s *Service) reportSummary(ctx context.Context) (ReportSummary, error) {
	sum := ReportSummary{ByStatus: map[string]int{}}
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM mortgage_applications
		WHERE assigned_advisor_id IS NULL AND status NOT IN ('CANCELLED','COMPLETED','REJECTED')
	`).Scan(&sum.UnassignedOpen)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM mortgage_applications WHERE status NOT IN ('CANCELLED','COMPLETED','REJECTED')
	`).Scan(&sum.OpenCases)
	rows, err := s.db.QueryContext(ctx, `
		SELECT status::text, COUNT(*) FROM mortgage_applications GROUP BY status
	`)
	if err != nil {
		return sum, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return sum, err
		}
		sum.ByStatus[st] = n
	}
	return sum, rows.Err()
}
