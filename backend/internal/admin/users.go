package admin

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
	"golang.org/x/crypto/bcrypt"
)

type adminUser struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	FullName   string    `json:"full_name"`
	LenderID   *string   `json:"lender_id,omitempty"`
	LenderName string    `json:"lender_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type createUserInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required,min=2"`
	Role     string `json:"role" binding:"required"`
	LenderID string `json:"lender_id"`
}

type updateUserInput struct {
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
	Status   *string `json:"status"`
	Password *string `json:"password"`
	LenderID *string `json:"lender_id"`
}

func (h *Handler) ListUsers(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT u.id::text, u.email, u.role::text, u.status, COALESCE(p.full_name,''), u.created_at,
			u.lender_id::text, COALESCE(l.name,'')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN lenders l ON l.id = u.lender_id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		httpx.Internal(c, "Could not load users.")
		return
	}
	defer rows.Close()
	out := []adminUser{}
	for rows.Next() {
		var it adminUser
		var lenderID sql.NullString
		if err := rows.Scan(&it.ID, &it.Email, &it.Role, &it.Status, &it.FullName, &it.CreatedAt, &lenderID, &it.LenderName); err != nil {
			httpx.Internal(c, "Could not load users.")
			return
		}
		if lenderID.Valid && lenderID.String != "" {
			s := lenderID.String
			it.LenderID = &s
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

func (h *Handler) CreateUser(c *gin.Context) {
	var in createUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Name, email, password (min 8 characters), and role are required.")
		return
	}
	email := normalizeEmail(in.Email)
	if !looksLikeEmail(email) {
		httpx.BadRequest(c, "Please provide a valid email.")
		return
	}
	if !validRole(in.Role) {
		httpx.BadRequest(c, "Role must be CUSTOMER, ADVISOR, ADMIN, or LENDER_USER.")
		return
	}
	lenderID, err := h.resolveLenderID(c.Request.Context(), in.Role, in.LenderID, nil)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	fullName := strings.TrimSpace(in.FullName)
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var id string
	err = tx.QueryRowContext(c.Request.Context(), `
		INSERT INTO users (email, password_hash, role, status, email_verified_at, lender_id)
		VALUES ($1, $2, $3, 'active', NOW(), $4)
		RETURNING id::text
	`, email, string(hash), in.Role, lenderID).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_unique") || strings.Contains(err.Error(), "duplicate") {
			httpx.Conflict(c, "An account with this email already exists.")
			return
		}
		httpx.Internal(c, "Could not create user.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO user_profiles (user_id, full_name) VALUES ($1::uuid, $2)`, id, fullName); err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO employment_profiles (user_id) VALUES ($1::uuid)`, id); err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO financial_profiles (user_id) VALUES ($1::uuid)`, id); err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.Internal(c, "Could not create user.")
		return
	}

	user, err := h.getUser(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "User created but could not be loaded.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown user.")
		return
	}
	actor, ok := actor(c)
	if !ok {
		httpx.Unauthorized(c, "Please sign in.")
		return
	}
	var in updateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Could not read those changes.")
		return
	}

	existing, err := h.getUser(c.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(c, "User not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not update user.")
		return
	}

	nextRole := existing.Role
	if in.Role != nil {
		if !validRole(*in.Role) {
			httpx.BadRequest(c, "Role must be CUSTOMER, ADVISOR, ADMIN, or LENDER_USER.")
			return
		}
		nextRole = *in.Role
	}
	nextStatus := existing.Status
	if in.Status != nil {
		if !validStatus(*in.Status) {
			httpx.BadRequest(c, "Status must be active or disabled.")
			return
		}
		nextStatus = *in.Status
	}

	if actor.ID.String() == id {
		if nextRole != "ADMIN" {
			httpx.Forbidden(c, "You cannot change your own role.")
			return
		}
		if nextStatus != "active" {
			httpx.Forbidden(c, "You cannot disable your own account.")
			return
		}
	}

	if wouldDropLastAdmin(existing, nextRole, nextStatus, false) {
		if err := h.guardLastAdmin(c.Request.Context(), id); err != nil {
			httpx.Conflict(c, "Keep at least one active admin on the platform.")
			return
		}
	}

	rawLender := ""
	if in.LenderID != nil {
		rawLender = *in.LenderID
	}
	lenderID, err := h.resolveLenderID(c.Request.Context(), nextRole, rawLender, existing.LenderID)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	if in.FullName != nil {
		name := strings.TrimSpace(*in.FullName)
		if len(name) < 2 {
			httpx.BadRequest(c, "Name must be at least 2 characters.")
			return
		}
		if _, err := h.db.ExecContext(c.Request.Context(), `
			INSERT INTO user_profiles (user_id, full_name) VALUES ($1::uuid, $2)
			ON CONFLICT (user_id) DO UPDATE SET full_name = EXCLUDED.full_name, updated_at = NOW()
		`, id, name); err != nil {
			httpx.Internal(c, "Could not update user.")
			return
		}
	}

	if _, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE users SET role=$2, status=$3, lender_id=$4::uuid, updated_at=NOW() WHERE id=$1::uuid AND deleted_at IS NULL
	`, id, nextRole, nextStatus, lenderID); err != nil {
		httpx.Internal(c, "Could not update user.")
		return
	}

	if in.Password != nil && strings.TrimSpace(*in.Password) != "" {
		if len(*in.Password) < 8 {
			httpx.BadRequest(c, "Password must be at least 8 characters.")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			httpx.Internal(c, "Could not update password.")
			return
		}
		if _, err := h.db.ExecContext(c.Request.Context(), `
			UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1::uuid
		`, id, string(hash)); err != nil {
			httpx.Internal(c, "Could not update password.")
			return
		}
	}

	user, err := h.getUser(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Could not load updated user.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown user.")
		return
	}
	actor, ok := actor(c)
	if !ok {
		httpx.Unauthorized(c, "Please sign in.")
		return
	}
	if actor.ID.String() == id {
		httpx.Forbidden(c, "You cannot remove your own account.")
		return
	}
	existing, err := h.getUser(c.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(c, "User not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not remove user.")
		return
	}
	if wouldDropLastAdmin(existing, existing.Role, "disabled", true) {
		if err := h.guardLastAdmin(c.Request.Context(), id); err != nil {
			httpx.Conflict(c, "Keep at least one active admin on the platform.")
			return
		}
	}
	if _, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE users SET deleted_at=NOW(), status='disabled', updated_at=NOW()
		WHERE id=$1::uuid AND deleted_at IS NULL
	`, id); err != nil {
		httpx.Internal(c, "Could not remove user.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) getUser(ctx context.Context, id string) (*adminUser, error) {
	var it adminUser
	var lenderID sql.NullString
	err := h.db.QueryRowContext(ctx, `
		SELECT u.id::text, u.email, u.role::text, u.status, COALESCE(p.full_name,''), u.created_at,
			u.lender_id::text, COALESCE(l.name,'')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN lenders l ON l.id = u.lender_id
		WHERE u.id=$1::uuid AND u.deleted_at IS NULL
	`, id).Scan(&it.ID, &it.Email, &it.Role, &it.Status, &it.FullName, &it.CreatedAt, &lenderID, &it.LenderName)
	if err != nil {
		return nil, err
	}
	if lenderID.Valid && lenderID.String != "" {
		s := lenderID.String
		it.LenderID = &s
	}
	return &it, nil
}

func (h *Handler) resolveLenderID(ctx context.Context, role, raw string, existing *string) (*string, error) {
	if role != "LENDER_USER" {
		return nil, nil
	}
	id := strings.TrimSpace(raw)
	if id == "" && existing != nil {
		id = strings.TrimSpace(*existing)
	}
	if id == "" {
		return nil, errors.New("Link a lender organisation for lender users.")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("Invalid lender.")
	}
	var n int
	if err := h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lenders WHERE id=$1 AND deleted_at IS NULL`, uid).Scan(&n); err != nil {
		return nil, errors.New("Could not check lender.")
	}
	if n == 0 {
		return nil, errors.New("Unknown lender.")
	}
	s := uid.String()
	return &s, nil
}

func (h *Handler) guardLastAdmin(ctx context.Context, targetID string) error {
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM users
		WHERE deleted_at IS NULL AND role='ADMIN' AND status='active' AND id <> $1::uuid
	`, targetID).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("last admin")
	}
	return nil
}

func wouldDropLastAdmin(existing *adminUser, nextRole, nextStatus string, deleting bool) bool {
	if existing.Role != "ADMIN" || existing.Status != "active" {
		return false
	}
	if deleting {
		return true
	}
	return nextRole != "ADMIN" || nextStatus != "active"
}

func actor(c *gin.Context) (auth.SessionUser, bool) {
	raw, ok := c.Get("auth_user")
	if !ok {
		return auth.SessionUser{}, false
	}
	su, ok := raw.(auth.SessionUser)
	return su, ok
}

func validRole(role string) bool {
	switch role {
	case "CUSTOMER", "ADVISOR", "ADMIN", "LENDER_USER":
		return true
	default:
		return false
	}
}

func validStatus(status string) bool {
	return status == "active" || status == "disabled"
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func looksLikeEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at < 1 || at != strings.LastIndexByte(email, '@') || at >= len(email)-3 {
		return false
	}
	local, domain := email[:at], email[at+1:]
	if strings.ContainsAny(local, " \t") || strings.ContainsAny(domain, " \t") {
		return false
	}
	for _, r := range email {
		if unicode.IsControl(r) {
			return false
		}
	}
	return strings.Contains(domain, ".")
}
