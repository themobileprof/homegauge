package applications

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

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterCustomer(rg *gin.RouterGroup) {
	rg.GET("/applications/me", h.Mine)
	rg.POST("/applications/request-advisor", h.RequestAdvisor)
}

func (h *Handler) RegisterAdvisor(rg *gin.RouterGroup) {
	rg.GET("/cases", h.ListCases)
	rg.GET("/cases/:id", h.GetCase)
	rg.PATCH("/cases/:id/status", h.UpdateStatus)
	rg.POST("/cases/:id/assign", h.Assign)
	rg.POST("/cases/:id/notes", h.AddNote)
	rg.GET("/cases/:id/notes", h.ListNotes)
	rg.GET("/cases/:id/suggestions", h.ListSuggestions)
	rg.POST("/suggestions/:id/resolve", h.ResolveSuggestion)
}

func (h *Handler) Mine(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	app, err := h.svc.GetMine(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "No active application yet.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load application.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app})
}

func (h *Handler) RequestAdvisor(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	app, err := h.svc.RequestAdvisor(c.Request.Context(), user.ID)
	if err != nil {
		httpx.Internal(c, "Could not request advisor.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app, "message": "An advisor has been notified."})
}

func (h *Handler) ListCases(c *gin.Context) {
	items, err := h.svc.ListCases(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "Could not load cases.")
		return
	}
	if items == nil {
		items = []Application{}
	}
	c.JSON(http.StatusOK, gin.H{"cases": items})
}

func (h *Handler) GetCase(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	app, err := h.svc.GetByID(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Case not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load case.")
		return
	}
	notes, _ := h.svc.ListNotes(c.Request.Context(), id)
	suggestions, _ := h.svc.ListSuggestions(c.Request.Context(), id)
	c.JSON(http.StatusOK, gin.H{"case": app, "notes": notes, "suggestions": suggestions})
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	var body struct {
		Status     string `json:"status" binding:"required"`
		NextAction string `json:"next_action_text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "status is required.")
		return
	}
	app, err := h.svc.UpdateStatus(c.Request.Context(), user.ID, id, body.Status, body.NextAction)
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, "Invalid status.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not update status.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"case": app})
}

func (h *Handler) Assign(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	var body struct {
		AdvisorID string `json:"advisor_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "advisor_id is required.")
		return
	}
	advisorID, err := uuid.Parse(body.AdvisorID)
	if err != nil {
		httpx.BadRequest(c, "Invalid advisor id.")
		return
	}
	if err := h.svc.Assign(c.Request.Context(), user.ID, id, advisorID); err != nil {
		httpx.Internal(c, "Could not assign advisor.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AddNote(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	var body struct {
		Body       string `json:"body" binding:"required"`
		Visibility string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "Note body is required.")
		return
	}
	note, err := h.svc.AddNote(c.Request.Context(), user.ID, id, body.Body, body.Visibility)
	if err != nil {
		httpx.Internal(c, "Could not add note.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"note": note})
}

func (h *Handler) ListNotes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	notes, err := h.svc.ListNotes(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Could not load notes.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *Handler) ListSuggestions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	items, err := h.svc.ListSuggestions(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Could not load suggestions.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": items})
}

func (h *Handler) ResolveSuggestion(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid suggestion id.")
		return
	}
	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "status is required.")
		return
	}
	if err := h.svc.ResolveSuggestion(c.Request.Context(), user.ID, id, body.Status); err != nil {
		httpx.BadRequest(c, "Could not resolve suggestion.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
