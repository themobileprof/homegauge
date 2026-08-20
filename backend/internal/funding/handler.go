package funding

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/webhooks/paystack", h.PaystackWebhook)
}

func (h *Handler) RegisterCustomer(rg *gin.RouterGroup) {
	rg.GET("/applications/me/funding", h.Mine)
	rg.POST("/applications/me/funding/account", h.EnsureAccount)
}

func (h *Handler) RegisterAdvisor(rg *gin.RouterGroup) {
	rg.GET("/cases/:id/funding", h.AdvisorGet)
	rg.PATCH("/cases/:id/funding/obligations/:oid", h.AdvisorPatchObligation)
}

func (h *Handler) Mine(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	snap, err := h.svc.SnapshotForUser(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "No active application yet. Complete eligibility and choose a product first.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load funding snapshot.")
		return
	}
	c.JSON(http.StatusOK, snap)
}

type ensureAccountBody struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

func (h *Handler) EnsureAccount(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	var body ensureAccountBody
	_ = c.ShouldBindJSON(&body)
	acc, err := h.svc.EnsureAccount(c.Request.Context(), user.ID, body.FirstName, body.LastName, body.Phone)
	if errors.Is(err, ErrDisabled) {
		httpx.BadRequest(c, "Paystack is not configured on this server yet.")
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "No active application yet.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, "Choose a mortgage product first, then open your case collection account.")
		return
	}
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": acc})
}

func (h *Handler) AdvisorGet(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	snap, err := h.svc.SnapshotForApp(c.Request.Context(), appID)
	if err != nil {
		httpx.Internal(c, "Could not load funding snapshot.")
		return
	}
	c.JSON(http.StatusOK, snap)
}

type patchObligationBody struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

func (h *Handler) AdvisorPatchObligation(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	oid, err := uuid.Parse(c.Param("oid"))
	if err != nil {
		httpx.BadRequest(c, "Invalid obligation id.")
		return
	}
	var body patchObligationBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "Provide a valid status.")
		return
	}
	o, err := h.svc.PatchObligation(c.Request.Context(), appID, oid, body.Status, body.Note)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Obligation not found.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, "Status must be pending, waived, or paid_offline.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not update obligation.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"obligation": o})
}

func (h *Handler) PaystackWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	sig := c.GetHeader("x-paystack-signature")
	if !h.svc.VerifyWebhookSignature(body, sig) {
		c.Status(http.StatusUnauthorized)
		return
	}
	if err := h.svc.HandleWebhook(c.Request.Context(), body); err != nil && !errors.Is(err, ErrNotFound) {
		httpx.Internal(c, "Webhook processing failed.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}
