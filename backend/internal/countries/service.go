package countries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrNotFound = errors.New("country not found")

type Country struct {
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	CurrencyCode  string          `json:"currency_code"`
	Locale        string          `json:"locale"`
	RegionLabel   string          `json:"region_label"`
	Regions       json.RawMessage `json:"regions"`
	DefaultITIPct float64         `json:"default_iti_pct"`
	Status        string          `json:"status"`
	SortOrder     int             `json:"sort_order"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, includeComingSoon bool) ([]Country, error) {
	q := `
		SELECT code, name, currency_code, locale, region_label, regions, default_iti_pct, status, sort_order
		FROM countries
		WHERE status = 'active'`
	if includeComingSoon {
		q = `
		SELECT code, name, currency_code, locale, region_label, regions, default_iti_pct, status, sort_order
		FROM countries
		WHERE status IN ('active', 'coming_soon')`
	}
	q += ` ORDER BY sort_order, name`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Country
	for rows.Next() {
		c, err := scanCountry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, code string) (*Country, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT code, name, currency_code, locale, region_label, regions, default_iti_pct, status, sort_order
		FROM countries WHERE code = $1
	`, code)
	c, err := scanCountry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) DefaultCode(ctx context.Context) (string, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT value FROM platform_settings WHERE key = 'default_country_code'`).Scan(&raw)
	if err == nil && len(raw) > 0 {
		var code string
		if json.Unmarshal(raw, &code) == nil && code != "" {
			return code, nil
		}
	}
	var code string
	err = s.db.QueryRowContext(ctx, `
		SELECT code FROM countries WHERE status = 'active' ORDER BY sort_order, name LIMIT 1
	`).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "NG", nil
	}
	return code, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanCountry(row scannable) (Country, error) {
	var c Country
	var regions []byte
	err := row.Scan(&c.Code, &c.Name, &c.CurrencyCode, &c.Locale, &c.RegionLabel, &regions, &c.DefaultITIPct, &c.Status, &c.SortOrder)
	if err != nil {
		return c, err
	}
	if len(regions) == 0 {
		c.Regions = json.RawMessage("[]")
	} else {
		c.Regions = regions
	}
	return c, nil
}
