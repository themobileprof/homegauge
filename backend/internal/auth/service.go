package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenInvalid       = errors.New("token invalid or expired")
	ErrEmailNotVerified   = errors.New("email not verified")
)

type Role string

const (
	RoleCustomer   Role = "CUSTOMER"
	RoleAdvisor    Role = "ADVISOR"
	RoleAdmin      Role = "ADMIN"
	RoleLenderUser Role = "LENDER_USER"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Role            Role       `json:"role"`
	LenderID        *uuid.UUID `json:"lender_id,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

type SessionUser struct {
	ID       uuid.UUID  `json:"id"`
	Email    string     `json:"email"`
	Role     Role       `json:"role"`
	LenderID *uuid.UUID `json:"lender_id,omitempty"`
}

type Service struct {
	db           *sql.DB
	rdb          *redis.Client
	mailer       Mailer
	appURL       string
	sessionTTL   time.Duration
	sessionKeyPx string
}

type Mailer interface {
	Send(to, subject, body string) error
}

func NewService(db *sql.DB, rdb *redis.Client, mailer Mailer, appURL string, sessionTTL time.Duration) *Service {
	return &Service{
		db:           db,
		rdb:          rdb,
		mailer:       mailer,
		appURL:       strings.TrimRight(appURL, "/"),
		sessionTTL:   sessionTTL,
		sessionKeyPx: "homegauge:session:",
	}
}

type RegisterInput struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FullName  string `json:"full_name" binding:"required,min=2"`
	Role      Role   `json:"-"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error) {
	email := normalizeEmail(in.Email)
	if in.Role == "" {
		in.Role = RoleCustomer
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var id uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`, email, string(hash), string(in.Role)).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_profiles (user_id, full_name) VALUES ($1, $2)
	`, id, strings.TrimSpace(in.FullName)); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO employment_profiles (user_id) VALUES ($1)
	`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO financial_profiles (user_id) VALUES ($1)
	`, id); err != nil {
		return nil, err
	}

	rawToken, tokenHash, err := newToken()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, purpose, expires_at)
		VALUES ($1, $2, 'email_verify', $3)
	`, id, tokenHash, time.Now().Add(48*time.Hour)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, rawToken)
	_ = s.mailer.Send(email, "Verify your HomeGauge email",
		fmt.Sprintf("Welcome to HomeGauge.\n\nConfirm your email:\n%s\n\nThis link expires in 48 hours.\n", verifyURL))

	return s.GetUser(ctx, id)
}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Service) Login(ctx context.Context, in LoginInput) (sessionID string, user *User, err error) {
	email := normalizeEmail(in.Email)
	var (
		id           uuid.UUID
		passwordHash string
		role         string
		status       string
		verifiedAt   sql.NullTime
		createdAt    time.Time
	)
	err = s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, role, status, email_verified_at, created_at
		FROM users
		WHERE LOWER(email) = $1 AND deleted_at IS NULL
	`, email).Scan(&id, &passwordHash, &role, &status, &verifiedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, err
	}
	if status != "active" {
		return "", nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(in.Password)) != nil {
		return "", nil, ErrInvalidCredentials
	}

	sessionID, err = randomID(32)
	if err != nil {
		return "", nil, err
	}
	su := SessionUser{ID: id, Email: email, Role: Role(role)}
	if err := s.rdb.Set(ctx, s.sessionKeyPx+sessionID, encodeSession(su), s.sessionTTL).Err(); err != nil {
		return "", nil, err
	}

	user = &User{
		ID:        id,
		Email:     email,
		Role:      Role(role),
		Status:    status,
		CreatedAt: createdAt,
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time
		user.EmailVerifiedAt = &t
	}
	return sessionID, user, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.rdb.Del(ctx, s.sessionKeyPx+sessionID).Err()
}

func (s *Service) SessionUser(ctx context.Context, sessionID string) (*SessionUser, error) {
	if sessionID == "" {
		return nil, ErrInvalidCredentials
	}
	val, err := s.rdb.Get(ctx, s.sessionKeyPx+sessionID).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	su, err := decodeSession(val)
	if err != nil {
		return nil, err
	}
	_ = s.rdb.Expire(ctx, s.sessionKeyPx+sessionID, s.sessionTTL).Err()
	return s.LiveSessionUser(ctx, su)
}

// LiveSessionUser applies current role/status from the database so admin
// disable, delete, and role changes take effect without waiting for re-login.
func (s *Service) LiveSessionUser(ctx context.Context, su SessionUser) (*SessionUser, error) {
	u, err := s.GetUser(ctx, su.ID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if u.Status != "active" {
		return nil, ErrInvalidCredentials
	}
	return &SessionUser{ID: u.ID, Email: u.Email, Role: u.Role, LenderID: u.LenderID}, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var (
		email      string
		role       string
		status     string
		verifiedAt sql.NullTime
		createdAt  time.Time
	)
	var lenderID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT email, role, status, email_verified_at, created_at, lender_id::text
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&email, &role, &status, &verifiedAt, &createdAt, &lenderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u := &User{ID: id, Email: email, Role: Role(role), Status: status, CreatedAt: createdAt}
	if lenderID.Valid && lenderID.String != "" {
		if lid, err := uuid.Parse(lenderID.String); err == nil {
			u.LenderID = &lid
		}
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time
		u.EmailVerifiedAt = &t
	}
	return u, nil
}

func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	userID, err := s.consumeToken(ctx, rawToken, "email_verify")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET email_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND email_verified_at IS NULL
	`, userID)
	return err
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	var userID uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM users WHERE LOWER(email) = $1 AND deleted_at IS NULL
	`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, hash, err := newToken()
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, purpose, expires_at)
		VALUES ($1, $2, 'password_reset', $3)
	`, userID, hash, time.Now().Add(2*time.Hour)); err != nil {
		return err
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, raw)
	return s.mailer.Send(email, "Reset your HomeGauge password",
		fmt.Sprintf("Reset your password:\n%s\n\nThis link expires in 2 hours. If you did not request this, ignore this email.\n", resetURL))
}

type ResetPasswordInput struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (s *Service) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	userID, err := s.consumeToken(ctx, in.Token, "password_reset")
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, string(hash))
	return err
}

func (s *Service) consumeToken(ctx context.Context, raw, purpose string) (uuid.UUID, error) {
	hash := hashToken(raw)
	var (
		id        uuid.UUID
		userID    uuid.UUID
		expiresAt time.Time
		usedAt    sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, used_at
		FROM auth_tokens
		WHERE token_hash = $1 AND purpose = $2
	`, hash, purpose).Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrTokenInvalid
	}
	if err != nil {
		return uuid.Nil, err
	}
	if usedAt.Valid || time.Now().After(expiresAt) {
		return uuid.Nil, ErrTokenInvalid
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE auth_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL
	`, id)
	if err != nil {
		return uuid.Nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return uuid.Nil, ErrTokenInvalid
	}
	return userID, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func newToken() (raw, hash string, err error) {
	raw, err = randomID(32)
	if err != nil {
		return "", "", err
	}
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func encodeSession(su SessionUser) string {
	return su.ID.String() + "|" + string(su.Role) + "|" + su.Email
}

func decodeSession(v string) (SessionUser, error) {
	parts := strings.SplitN(v, "|", 3)
	if len(parts) != 3 {
		return SessionUser{}, ErrInvalidCredentials
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return SessionUser{}, ErrInvalidCredentials
	}
	return SessionUser{ID: id, Role: Role(parts[1]), Email: parts[2]}, nil
}
