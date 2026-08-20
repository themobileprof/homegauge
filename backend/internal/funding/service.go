package funding

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/paystack"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrDisabled  = errors.New("paystack disabled")
	ErrInvalid   = errors.New("invalid")
)

type Service struct {
	db        *sql.DB
	ps        *paystack.Client
	secretKey string
}

func NewService(db *sql.DB, ps *paystack.Client, secretKey string) *Service {
	return &Service{db: db, ps: ps, secretKey: strings.TrimSpace(secretKey)}
}

func (s *Service) Enabled() bool {
	return s.ps != nil && s.ps.Enabled()
}

type Account struct {
	ID                   string    `json:"id"`
	ApplicationID        string    `json:"application_id"`
	PaystackCustomerCode string    `json:"paystack_customer_code"`
	AccountNumber        string    `json:"account_number"`
	AccountName          string    `json:"account_name"`
	BankName             string    `json:"bank_name"`
	BankSlug             string    `json:"bank_slug"`
	CurrencyCode         string    `json:"currency_code"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
}

type Obligation struct {
	ID             string   `json:"id"`
	ApplicationID  string   `json:"application_id"`
	ObligationKey  string   `json:"obligation_key"`
	Label          string   `json:"label"`
	Amount         *float64 `json:"amount"`
	AmountReceived float64  `json:"amount_received"`
	CurrencyCode   string   `json:"currency_code"`
	DuePhase       string   `json:"due_phase"`
	Collectable    bool     `json:"collectable"`
	Status         string   `json:"status"`
	Note           string   `json:"note"`
	SortOrder      int      `json:"sort_order"`
}

type Movement struct {
	ID                string    `json:"id"`
	ApplicationID     string    `json:"application_id"`
	Direction         string    `json:"direction"`
	Amount            float64   `json:"amount"`
	CurrencyCode      string    `json:"currency_code"`
	PaystackReference string    `json:"paystack_reference"`
	PaystackEvent     string    `json:"paystack_event"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

type Snapshot struct {
	Enabled           bool         `json:"enabled"`
	PaystackPublicKey string       `json:"paystack_public_key,omitempty"`
	Account           *Account     `json:"account"`
	Obligations       []Obligation `json:"obligations"`
	Movements         []Movement   `json:"movements"`
	TotalDue          float64      `json:"total_due"`
	TotalReceived     float64      `json:"total_received"`
	TotalOutstanding  float64      `json:"total_outstanding"`
	AllSettled        bool         `json:"all_settled"`
	PreferredProduct  string       `json:"preferred_product_name,omitempty"`
}

func (s *Service) SnapshotForUser(ctx context.Context, userID uuid.UUID) (*Snapshot, error) {
	appID, productID, productName, err := s.userApp(ctx, userID)
	if err != nil {
		return nil, err
	}
	if productID != nil {
		_ = s.SyncObligations(ctx, appID, *productID)
	}
	return s.snapshot(ctx, appID, productName)
}

func (s *Service) SnapshotForApp(ctx context.Context, appID uuid.UUID) (*Snapshot, error) {
	var productID sql.NullString
	var productName string
	_ = s.db.QueryRowContext(ctx, `
		SELECT a.preferred_product_id::text, COALESCE(p.name,'')
		FROM mortgage_applications a
		LEFT JOIN mortgage_products p ON p.id = a.preferred_product_id
		WHERE a.id=$1
	`, appID).Scan(&productID, &productName)
	if productID.Valid {
		if pid, err := uuid.Parse(productID.String); err == nil {
			_ = s.SyncObligations(ctx, appID, pid)
		}
	}
	return s.snapshot(ctx, appID, productName)
}

func (s *Service) snapshot(ctx context.Context, appID uuid.UUID, productName string) (*Snapshot, error) {
	acc, _ := s.getAccount(ctx, appID)
	obs, err := s.listObligations(ctx, appID)
	if err != nil {
		return nil, err
	}
	movs, err := s.listMovements(ctx, appID)
	if err != nil {
		return nil, err
	}
	var due, received float64
	hasCollectable := false
	allSettled := true
	for _, o := range obs {
		received += o.AmountReceived
		if !o.Collectable || o.Amount == nil || *o.Amount <= 0 {
			continue
		}
		hasCollectable = true
		due += *o.Amount
		if o.Status != "paid" && o.Status != "waived" && o.Status != "paid_offline" {
			allSettled = false
		}
	}
	if !hasCollectable {
		allSettled = false
	}
	out := &Snapshot{
		Enabled:          s.Enabled(),
		Account:          acc,
		Obligations:      obs,
		Movements:        movs,
		TotalDue:         due,
		TotalReceived:    received,
		TotalOutstanding: max0(due - received),
		AllSettled:       allSettled,
		PreferredProduct: productName,
	}
	if s.ps != nil {
		out.PaystackPublicKey = s.ps.PublicKey()
	}
	return out, nil
}

func max0(n float64) float64 {
	if n < 0 {
		return 0
	}
	return n
}

func (s *Service) userApp(ctx context.Context, userID uuid.UUID) (uuid.UUID, *uuid.UUID, string, error) {
	var appID uuid.UUID
	var product sql.NullString
	var name string
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.preferred_product_id::text, COALESCE(p.name,'')
		FROM mortgage_applications a
		LEFT JOIN mortgage_products p ON p.id = a.preferred_product_id
		WHERE a.user_id=$1 AND a.status NOT IN ('CANCELLED')
		ORDER BY a.updated_at DESC LIMIT 1
	`, userID).Scan(&appID, &product, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, "", ErrNotFound
	}
	if err != nil {
		return uuid.Nil, nil, "", err
	}
	if !product.Valid {
		return appID, nil, name, nil
	}
	pid, err := uuid.Parse(product.String)
	if err != nil {
		return appID, nil, name, nil
	}
	return appID, &pid, name, nil
}

func (s *Service) SyncObligations(ctx context.Context, appID, productID uuid.UUID) error {
	var processing, valuation, legal, equity sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT processing_fee, valuation_fee, legal_fee, min_equity_pct
		FROM mortgage_products WHERE id=$1 AND deleted_at IS NULL
	`, productID).Scan(&processing, &valuation, &legal, &equity)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	type seed struct {
		key, label, phase string
		amount           *float64
		collectable      bool
		note             string
		order            int
	}
	seeds := []seed{
		{key: "equity", label: "Equity / deposit", phase: "before_disbursement", collectable: false, order: 10, note: noteEquity(equity)},
		{key: "valuation", label: "Valuation fee", phase: "before_approval", collectable: true, order: 20, amount: nullFloat(valuation), note: "Paid toward valuation via this case collection account when possible."},
		{key: "legal", label: "Legal / search fees", phase: "before_approval", collectable: true, order: 30, amount: nullFloat(legal), note: "Title/legal search and related pre-disbursement legal costs."},
		{key: "processing", label: "Processing / originating fee", phase: "at_offer", collectable: true, order: 40, amount: nullFloat(processing), note: "Often due at offer acceptance — still pre-disbursement."},
		{key: "repayment_setup", label: "Repayment mandate setup", phase: "before_disbursement", collectable: false, order: 50, note: "Salary domicile and/or GSI/direct debit with the mortgage bank — confirm offline; not collected here."},
	}

	for _, item := range seeds {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO funding_obligations (
				application_id, obligation_key, label, amount, currency_code, due_phase, collectable, note, sort_order, status
			) VALUES ($1,$2,$3,$4,'NGN',$5,$6,$7,$8,'pending')
			ON CONFLICT (application_id, obligation_key) DO UPDATE SET
				label = EXCLUDED.label,
				amount = COALESCE(funding_obligations.amount, EXCLUDED.amount),
				due_phase = EXCLUDED.due_phase,
				collectable = EXCLUDED.collectable,
				note = CASE WHEN funding_obligations.note = '' THEN EXCLUDED.note ELSE funding_obligations.note END,
				sort_order = EXCLUDED.sort_order,
				updated_at = NOW()
		`, appID, item.key, item.label, item.amount, item.phase, item.collectable, item.note, item.order)
		if err != nil {
			return err
		}
	}
	return s.refreshObligationStatuses(ctx, appID)
}

func noteEquity(equity sql.NullFloat64) string {
	if equity.Valid {
		return fmt.Sprintf("Plan for about %.0f%% of the property price before disbursement. Confirm amount with advisor; usually not paid into this fee account.", equity.Float64)
	}
	return "Confirm equity percentage and amount with your advisor and lender."
}

func nullFloat(n sql.NullFloat64) *float64 {
	if !n.Valid || n.Float64 <= 0 {
		return nil
	}
	v := n.Float64
	return &v
}

func (s *Service) EnsureAccount(ctx context.Context, userID uuid.UUID, firstName, lastName, phone string) (*Account, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	appID, productID, _, err := s.userApp(ctx, userID)
	if err != nil {
		return nil, err
	}
	if productID == nil {
		return nil, ErrInvalid
	}
	_ = s.SyncObligations(ctx, appID, *productID)

	if existing, err := s.getAccount(ctx, appID); err == nil && existing != nil && existing.AccountNumber != "" {
		return existing, nil
	}

	var email string
	_ = s.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&email)
	if email == "" {
		return nil, ErrInvalid
	}
	if firstName == "" {
		var full string
		_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(full_name,'') FROM user_profiles WHERE user_id=$1`, userID).Scan(&full)
		firstName, lastName = splitName(full)
	}
	if firstName == "" {
		firstName = "Homebuyer"
	}
	if lastName == "" {
		lastName = "HomeGauge"
	}

	acc, err := s.ps.AssignDedicatedAccount(ctx, email, firstName, lastName, phone, s.ps.DVABank(), "NG")
	if err != nil {
		cust, cerr := s.ps.CreateCustomer(ctx, email, firstName, lastName, phone)
		if cerr != nil {
			return nil, fmt.Errorf("%v; fallback: %v", err, cerr)
		}
		acc, err = s.ps.CreateDedicatedAccount(ctx, cust.CustomerCode, s.ps.DVABank(), firstName, lastName, phone)
		if err != nil {
			return nil, err
		}
		if acc.Customer.CustomerCode == "" {
			acc.Customer.CustomerCode = cust.CustomerCode
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO funding_accounts (
			application_id, user_id, paystack_customer_code, paystack_dva_id,
			account_number, account_name, bank_name, bank_slug, currency_code, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,COALESCE(NULLIF($9,''),'NGN'),'active')
		ON CONFLICT (application_id) DO UPDATE SET
			paystack_customer_code = EXCLUDED.paystack_customer_code,
			paystack_dva_id = EXCLUDED.paystack_dva_id,
			account_number = EXCLUDED.account_number,
			account_name = EXCLUDED.account_name,
			bank_name = EXCLUDED.bank_name,
			bank_slug = EXCLUDED.bank_slug,
			currency_code = EXCLUDED.currency_code,
			status = 'active',
			updated_at = NOW()
	`, appID, userID, acc.Customer.CustomerCode, acc.ID, acc.AccountNumber, acc.AccountName, acc.Bank.Name, acc.Bank.Slug, acc.Currency)
	if err != nil {
		return nil, err
	}
	return s.getAccount(ctx, appID)
}

func splitName(full string) (string, string) {
	full = strings.TrimSpace(full)
	if full == "" {
		return "", ""
	}
	parts := strings.Fields(full)
	if len(parts) == 1 {
		return parts[0], "HomeGauge"
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func (s *Service) getAccount(ctx context.Context, appID uuid.UUID) (*Account, error) {
	var a Account
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, application_id::text, paystack_customer_code, account_number, account_name,
			bank_name, bank_slug, currency_code, status, created_at
		FROM funding_accounts WHERE application_id=$1
	`, appID).Scan(
		&a.ID, &a.ApplicationID, &a.PaystackCustomerCode, &a.AccountNumber, &a.AccountName,
		&a.BankName, &a.BankSlug, &a.CurrencyCode, &a.Status, &a.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) listObligations(ctx context.Context, appID uuid.UUID) ([]Obligation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, application_id::text, obligation_key, label, amount, amount_received,
			currency_code, due_phase, collectable, status, note, sort_order
		FROM funding_obligations WHERE application_id=$1 ORDER BY sort_order, created_at
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Obligation{}
	for rows.Next() {
		var o Obligation
		var amount sql.NullFloat64
		if err := rows.Scan(
			&o.ID, &o.ApplicationID, &o.ObligationKey, &o.Label, &amount, &o.AmountReceived,
			&o.CurrencyCode, &o.DuePhase, &o.Collectable, &o.Status, &o.Note, &o.SortOrder,
		); err != nil {
			return nil, err
		}
		if amount.Valid {
			v := amount.Float64
			o.Amount = &v
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Service) listMovements(ctx context.Context, appID uuid.UUID) ([]Movement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, application_id::text, direction, amount, currency_code,
			paystack_reference, paystack_event, status, created_at
		FROM funding_movements WHERE application_id=$1 ORDER BY created_at DESC LIMIT 50
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(
			&m.ID, &m.ApplicationID, &m.Direction, &m.Amount, &m.CurrencyCode,
			&m.PaystackReference, &m.PaystackEvent, &m.Status, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) PatchObligation(ctx context.Context, appID, obligationID uuid.UUID, status, note string) (*Obligation, error) {
	switch status {
	case "waived", "paid_offline", "pending":
	default:
		return nil, ErrInvalid
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE funding_obligations
		SET status=$3,
			note = CASE WHEN NULLIF($4,'') IS NULL THEN note ELSE $4 END,
			updated_at=NOW()
		WHERE id=$1 AND application_id=$2
	`, obligationID, appID, status, note)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	_ = s.refreshObligationStatuses(ctx, appID)
	obs, err := s.listObligations(ctx, appID)
	if err != nil {
		return nil, err
	}
	for i := range obs {
		if obs[i].ID == obligationID.String() {
			return &obs[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) VerifyWebhookSignature(body []byte, signature string) bool {
	if s.secretKey == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha512.New, []byte(s.secretKey))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

type webhookEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type chargeData struct {
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Reference string `json:"reference"`
	Status    string `json:"status"`
	Customer  struct {
		CustomerCode string `json:"customer_code"`
		Email        string `json:"email"`
	} `json:"customer"`
	Authorization struct {
		ReceiverBankAccountNumber string `json:"receiver_bank_account_number"`
		ReceiverBank              string `json:"receiver_bank"`
	} `json:"authorization"`
}

func (s *Service) HandleWebhook(ctx context.Context, body []byte) error {
	var env webhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return ErrInvalid
	}
	switch env.Event {
	case "charge.success", "dedicatedaccount.assign.success":
		// continue for charge.success mainly
	default:
		return nil
	}
	if env.Event != "charge.success" {
		return nil
	}
	var data chargeData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return ErrInvalid
	}
	if !strings.EqualFold(data.Status, "success") {
		return nil
	}
	if data.Reference == "" || data.Amount <= 0 {
		return nil
	}

	appID, err := s.resolveAppForCharge(ctx, data)
	if err != nil {
		return err
	}
	amountNGN := float64(data.Amount) / 100.0
	currency := data.Currency
	if currency == "" {
		currency = "NGN"
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO funding_movements (
			application_id, direction, amount, currency_code, paystack_reference, paystack_event, status, payload
		) VALUES ($1,'credit',$2,$3,$4,$5,'success',$6::jsonb)
		ON CONFLICT (paystack_reference) DO NOTHING
	`, appID, amountNGN, currency, data.Reference, env.Event, string(body))
	if err != nil {
		return err
	}
	return s.allocateCredit(ctx, appID, amountNGN)
}

func (s *Service) resolveAppForCharge(ctx context.Context, data chargeData) (uuid.UUID, error) {
	var appID uuid.UUID
	if data.Authorization.ReceiverBankAccountNumber != "" {
		err := s.db.QueryRowContext(ctx, `
			SELECT application_id FROM funding_accounts WHERE account_number=$1 LIMIT 1
		`, data.Authorization.ReceiverBankAccountNumber).Scan(&appID)
		if err == nil {
			return appID, nil
		}
	}
	if data.Customer.CustomerCode != "" {
		err := s.db.QueryRowContext(ctx, `
			SELECT application_id FROM funding_accounts WHERE paystack_customer_code=$1 LIMIT 1
		`, data.Customer.CustomerCode).Scan(&appID)
		if err == nil {
			return appID, nil
		}
	}
	if data.Customer.Email != "" {
		err := s.db.QueryRowContext(ctx, `
			SELECT a.id
			FROM mortgage_applications a
			JOIN users u ON u.id = a.user_id
			WHERE LOWER(u.email)=LOWER($1) AND a.status NOT IN ('CANCELLED')
			ORDER BY a.updated_at DESC LIMIT 1
		`, data.Customer.Email).Scan(&appID)
		if err == nil {
			return appID, nil
		}
	}
	return uuid.Nil, ErrNotFound
}

func (s *Service) allocateCredit(ctx context.Context, appID uuid.UUID, amount float64) error {
	obs, err := s.listObligations(ctx, appID)
	if err != nil {
		return err
	}
	remaining := amount
	for _, o := range obs {
		if remaining <= 0 {
			break
		}
		if !o.Collectable || o.Amount == nil || *o.Amount <= 0 {
			continue
		}
		if o.Status == "waived" || o.Status == "paid_offline" || o.Status == "paid" {
			continue
		}
		need := *o.Amount - o.AmountReceived
		if need <= 0 {
			continue
		}
		apply := need
		if apply > remaining {
			apply = remaining
		}
		_, err := s.db.ExecContext(ctx, `
			UPDATE funding_obligations
			SET amount_received = amount_received + $2, updated_at=NOW()
			WHERE id=$1::uuid
		`, o.ID, apply)
		if err != nil {
			return err
		}
		remaining -= apply
	}
	return s.refreshObligationStatuses(ctx, appID)
}

func (s *Service) refreshObligationStatuses(ctx context.Context, appID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE funding_obligations
		SET status = CASE
			WHEN status IN ('waived','paid_offline') THEN status
			WHEN collectable = FALSE THEN status
			WHEN amount IS NULL OR amount <= 0 THEN status
			WHEN amount_received >= amount THEN 'paid'
			WHEN amount_received > 0 THEN 'partial'
			ELSE 'pending'
		END,
		updated_at = NOW()
		WHERE application_id=$1
	`, appID)
	return err
}
