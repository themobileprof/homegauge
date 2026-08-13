package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

const SessionCookie = "homegauge_session"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/logout", h.Logout)
	rg.POST("/verify-email", h.VerifyEmail)
	rg.POST("/forgot-password", h.ForgotPassword)
	rg.POST("/reset-password", h.ResetPassword)
}

func (h *Handler) RegisterAuthenticated(rg *gin.RouterGroup) {
	rg.GET("/me", h.Me)
}

func (h *Handler) Register(c *gin.Context) {
	var in RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Please provide a valid email, password (min 8 characters), and full name.")
		return
	}
	user, err := h.svc.Register(c.Request.Context(), in)
	if errors.Is(err, ErrEmailTaken) {
		httpx.Conflict(c, "An account with this email already exists.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not create account.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":    user,
		"message": "Account created. Please verify your email to continue.",
	})
}

func (h *Handler) Login(c *gin.Context) {
	var in LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Please provide email and password.")
		return
	}
	sessionID, user, err := h.svc.Login(c.Request.Context(), in)
	if errors.Is(err, ErrInvalidCredentials) {
		httpx.Unauthorized(c, "Incorrect email or password.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not sign in.")
		return
	}
	setSessionCookie(c, sessionID, h.svc.sessionTTL)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) Logout(c *gin.Context) {
	sid, _ := c.Cookie(SessionCookie)
	_ = h.svc.Logout(c.Request.Context(), sid)
	c.SetCookie(SessionCookie, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Me(c *gin.Context) {
	su, ok := c.Get("auth_user")
	if !ok {
		httpx.Unauthorized(c, "Not signed in.")
		return
	}
	sessionUser := su.(SessionUser)
	user, err := h.svc.GetUser(c.Request.Context(), sessionUser.ID)
	if err != nil {
		httpx.Unauthorized(c, "Not signed in.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "Verification token is required.")
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), body.Token); err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			httpx.BadRequest(c, "This verification link is invalid or has expired.")
			return
		}
		httpx.Internal(c, "Could not verify email.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email verified. You can continue."})
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "A valid email is required.")
		return
	}
	if err := h.svc.RequestPasswordReset(c.Request.Context(), body.Email); err != nil {
		httpx.Internal(c, "Could not start password reset.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "If an account exists for that email, reset instructions were sent."})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var in ResetPasswordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Token and a new password (min 8 characters) are required.")
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), in); err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			httpx.BadRequest(c, "This reset link is invalid or has expired.")
			return
		}
		httpx.Internal(c, "Could not reset password.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated. You can sign in."})
}

func setSessionCookie(c *gin.Context, sessionID string, ttl interface{ Seconds() float64 }) {
	maxAge := int(ttl.Seconds())
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookie, sessionID, maxAge, "/", "", secure, true)
}
