package eligibility

import (
	"errors"
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/assessments", h.Start)
	rg.GET("/assessments/latest", h.Latest)
	rg.GET("/assessments/:id", h.Get)
	rg.PATCH("/assessments/:id", h.Update)
	rg.POST("/assessments/:id/complete", h.Complete)
}

func (h *Handler) Start(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	a, err := h.svc.Start(c.Request.Context(), user.ID)
	if err != nil {
		httpx.Internal(c, "Could not start assessment.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"assessment": a})
}

func (h *Handler) Latest(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	a, err := h.svc.LatestForUser(c.Request.Context(), user.ID)
	if errors.Is(err, ErrAssessmentNotFound) {
		httpx.NotFound(c, "No assessment yet.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load assessment.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"assessment": a})
}

func (h *Handler) Get(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid assessment id.")
		return
	}
	a, err := h.svc.Get(c.Request.Context(), user.ID, id)
	if errors.Is(err, ErrAssessmentNotFound) {
		httpx.NotFound(c, "Assessment not found.")
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "You cannot view this assessment.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load assessment.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"assessment": a})
}

func (h *Handler) Update(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid assessment id.")
		return
	}
	var patch AssessmentInput
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.BadRequest(c, "Invalid assessment data.")
		return
	}
	a, err := h.svc.UpdateStep(c.Request.Context(), user.ID, id, patch)
	if errors.Is(err, ErrAssessmentNotFound) {
		httpx.NotFound(c, "Assessment not found.")
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "You cannot update this assessment.")
		return
	}
	if errors.Is(err, ErrInvalidStep) {
		httpx.BadRequest(c, "This assessment is already completed.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not save assessment step.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"assessment": a})
}

func (h *Handler) Complete(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid assessment id.")
		return
	}
	a, err := h.svc.Complete(c.Request.Context(), user.ID, id)
	if errors.Is(err, ErrAssessmentNotFound) {
		httpx.NotFound(c, "Assessment not found.")
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "You cannot complete this assessment.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not complete assessment.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"assessment": a,
		"disclaimer": "Based on the information provided, results show how you appear against stated product criteria. This is not a bank approval.",
	})
}
