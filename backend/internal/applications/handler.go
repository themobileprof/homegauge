package applications

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/documents"
	"github.com/homegauge/homegauge/backend/internal/eligibility"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct {
	svc  *Service
	docs *documents.Service
	elig *eligibility.Service
}

func NewHandler(svc *Service, docs *documents.Service, elig *eligibility.Service) *Handler {
	return &Handler{svc: svc, docs: docs, elig: elig}
}

func (h *Handler) RegisterCustomer(rg *gin.RouterGroup) {
	rg.GET("/applications/me", h.Mine)
	rg.POST("/applications/request-advisor", h.RequestAdvisor)
	rg.PATCH("/applications/me/product", h.SetMyProduct)
}

func (h *Handler) RegisterAdvisor(rg *gin.RouterGroup) {
	rg.GET("/cases", h.ListCases)
	rg.GET("/cases/:id", h.GetCase)
	rg.PATCH("/cases/:id/status", h.UpdateStatus)
	rg.PATCH("/cases/:id/product", h.AdvisorSetProduct)
	rg.POST("/cases/:id/notes", h.AddNote)
	rg.GET("/cases/:id/notes", h.ListNotes)
	rg.GET("/cases/:id/suggestions", h.ListSuggestions)
	rg.POST("/suggestions/:id/resolve", h.ResolveSuggestion)
}

func (h *Handler) RegisterLender(rg *gin.RouterGroup) {
	rg.GET("/me", h.LenderMe)
	rg.GET("/pipeline", h.LenderPipeline)
	rg.GET("/cases/:id", h.LenderGetCase)
	rg.PATCH("/cases/:id/status", h.LenderUpdateStatus)
	rg.POST("/cases/:id/notes", h.LenderAddNote)
}

func (h *Handler) RegisterAdmin(rg *gin.RouterGroup) {
	rg.GET("/cases", h.AdminListCases)
	rg.GET("/cases/:id", h.AdminGetCase)
	rg.PATCH("/cases/:id/status", h.AdminUpdateStatus)
	rg.POST("/cases/:id/assign", h.AdminAssign)
	rg.GET("/advisors", h.ListAdvisors)
	rg.GET("/reports/advisors", h.ReportAdvisors)
	rg.GET("/reports/buyers", h.ReportBuyers)
	rg.GET("/approvals", h.ListApprovals)
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
	docs := []documents.ChecklistItem{}
	if h.docs != nil {
		if items, _, err := h.docs.Checklist(c.Request.Context(), user.ID, app.PreferredProductID); err == nil {
			docs = items
		}
	}
	notes, _ := h.svc.ListNotesVisible(c.Request.Context(), app.ID, []string{"customer"})
	c.JSON(http.StatusOK, gin.H{"application": app, "documents": docs, "notes": notes})
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

func (h *Handler) SetMyProduct(c *gin.Context) {
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
	productID, ok := parseProductBody(c)
	if !ok {
		return
	}
	app, err = h.svc.SetPreferredProduct(c.Request.Context(), user.ID, app.ID, productID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Product not found.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, "That product is not available.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not save product choice.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app})
}

func (h *Handler) ListCases(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	items, err := h.svc.ListAdvisorCases(c.Request.Context(), user.ID)
	if err != nil {
		httpx.Internal(c, "Could not load cases.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"cases": items})
}

func (h *Handler) GetCase(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	app, err := h.svc.GetAdvisorCase(c.Request.Context(), user.ID, id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Case not found.")
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "This case is not assigned to you.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load case.")
		return
	}
	notes, _ := h.svc.ListNotes(c.Request.Context(), id)
	suggestions, _ := h.svc.ListSuggestions(c.Request.Context(), id)
	docs := []documents.ChecklistItem{}
	if h.docs != nil {
		if items, _, err := h.docs.Checklist(c.Request.Context(), app.UserID, app.PreferredProductID); err == nil {
			docs = items
		}
	}
	var assessment *eligibility.Assessment
	if h.elig != nil {
		if app.AssessmentID != nil {
			if a, err := h.elig.Get(c.Request.Context(), app.UserID, *app.AssessmentID); err == nil {
				assessment = a
			}
		} else if a, err := h.elig.LatestForUser(c.Request.Context(), app.UserID); err == nil {
			assessment = a
		}
	}
	c.JSON(http.StatusOK, gin.H{"case": app, "notes": notes, "suggestions": suggestions, "documents": docs, "assessment": assessment})
}

func (h *Handler) AdvisorSetProduct(c *gin.Context) {
	if !h.requireAssigned(c) {
		return
	}
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	productID, ok := parseProductBody(c)
	if !ok {
		return
	}
	app, err := h.svc.SetPreferredProduct(c.Request.Context(), user.ID, id, productID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Product not found.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, "That product is not available.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not save product.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"case": app})
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	if !h.requireAssigned(c) {
		return
	}
	h.updateStatus(c, advisorWorkingStatuses, "Advisors can update working status only. An admin sets approved, rejected, completed, or cancelled.")
}

func (h *Handler) AdminUpdateStatus(c *gin.Context) {
	h.updateStatus(c, allCaseStatuses, "Invalid status.")
}

func (h *Handler) updateStatus(c *gin.Context, allowed map[string]bool, invalidMsg string) {
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
	if body.Status == "SUBMITTED_TO_LENDER" || body.Status == "LENDER_REVIEW" {
		cur, err := h.svc.GetByID(c.Request.Context(), id)
		if err != nil {
			httpx.Internal(c, "Could not update status.")
			return
		}
		if cur.PreferredProductID == nil {
			httpx.BadRequest(c, "Choose a product (and lender) before sending this file.")
			return
		}
	}
	app, err := h.svc.UpdateStatus(c.Request.Context(), user.ID, id, body.Status, body.NextAction, allowed)
	if errors.Is(err, ErrInvalid) {
		httpx.BadRequest(c, invalidMsg)
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not update status.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"case": app})
}

func (h *Handler) requireAssigned(c *gin.Context) bool {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return false
	}
	_, err = h.svc.GetAdvisorCase(c.Request.Context(), user.ID, id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Case not found.")
		return false
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "This case is not assigned to you.")
		return false
	}
	if err != nil {
		httpx.Internal(c, "Could not load case.")
		return false
	}
	return true
}

func (h *Handler) AddNote(c *gin.Context) {
	if !h.requireAssigned(c) {
		return
	}
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
	if !h.requireAssigned(c) {
		return
	}
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
	if !h.requireAssigned(c) {
		return
	}
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

func (h *Handler) AdminListCases(c *gin.Context) {
	q := CaseQuery{IncludeClosed: true, Limit: 200}
	if c.Query("unassigned") == "1" || c.Query("assigned") == "unassigned" {
		q.Unassigned = true
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		q.Status = status
	}
	if raw := strings.TrimSpace(c.Query("advisor_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.BadRequest(c, "Invalid advisor id.")
			return
		}
		q.AssignedTo = &id
	}
	items, err := h.svc.ListCases(c.Request.Context(), q)
	if err != nil {
		httpx.Internal(c, "Could not load cases.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"cases": items})
}

func (h *Handler) AdminGetCase(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{"case": app, "notes": notes})
}

func (h *Handler) AdminAssign(c *gin.Context) {
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
	app, err := h.svc.Assign(c.Request.Context(), user.ID, id, advisorID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Case not found.")
		return
	}
	if errors.Is(err, ErrInvalidAdvisor) {
		httpx.BadRequest(c, "Choose an active advisor.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not assign advisor.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"case": app})
}

func (h *Handler) ListAdvisors(c *gin.Context) {
	items, err := h.svc.ListActiveAdvisors(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "Could not load advisors.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"advisors": items})
}

func (h *Handler) ReportAdvisors(c *gin.Context) {
	rows, sum, err := h.svc.ReportAdvisors(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "Could not load advisor report.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"advisors": rows, "summary": sum})
}

func (h *Handler) ReportBuyers(c *gin.Context) {
	rows, sum, err := h.svc.ReportBuyers(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "Could not load homebuyer report.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"buyers": rows, "summary": sum})
}

func (h *Handler) ListApprovals(c *gin.Context) {
	items, err := h.svc.ListCases(c.Request.Context(), CaseQuery{
		Statuses: []string{"READY_FOR_SUBMISSION"},
		Limit:    100,
	})
	if err != nil {
		httpx.Internal(c, "Could not load approvals.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"cases": items,
		"note":  "When an advisor marks a case ready for submission, it appears here for a top-level status decision. Exception and lender-offer approvals will be added later.",
	})
}

func parseProductBody(c *gin.Context) (uuid.UUID, bool) {
	var body struct {
		ProductID string `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "product_id is required.")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(body.ProductID)
	if err != nil {
		httpx.BadRequest(c, "Invalid product id.")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) requireLender(c *gin.Context) (auth.SessionUser, uuid.UUID, bool) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	if user.LenderID == nil {
		httpx.Forbidden(c, "Your account is not linked to a lender. Ask an admin to attach it.")
		return user, uuid.Nil, false
	}
	return user, *user.LenderID, true
}

func (h *Handler) LenderMe(c *gin.Context) {
	user, lenderID, ok := h.requireLender(c)
	if !ok {
		return
	}
	org, err := h.svc.LenderOrg(c.Request.Context(), lenderID)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Lender organisation not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load lender.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"lender": org, "email": user.Email})
}

func (h *Handler) LenderPipeline(c *gin.Context) {
	_, lenderID, ok := h.requireLender(c)
	if !ok {
		return
	}
	items, err := h.svc.ListLenderCases(c.Request.Context(), lenderID)
	if err != nil {
		httpx.Internal(c, "Could not load pipeline.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"cases": items})
}

func (h *Handler) LenderGetCase(c *gin.Context) {
	_, lenderID, ok := h.requireLender(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	app, err := h.svc.GetLenderCase(c.Request.Context(), lenderID, id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Case not found.")
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpx.Forbidden(c, "This file is not in your pipeline.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load case.")
		return
	}
	notes, _ := h.svc.ListNotesVisible(c.Request.Context(), id, []string{"customer", "lender"})
	docs := []documents.ChecklistItem{}
	if h.docs != nil {
		if items, _, err := h.docs.Checklist(c.Request.Context(), app.UserID, app.PreferredProductID); err == nil {
			docs = items
		}
	}
	var assessment *eligibility.Assessment
	if h.elig != nil {
		if app.AssessmentID != nil {
			if a, err := h.elig.Get(c.Request.Context(), app.UserID, *app.AssessmentID); err == nil {
				assessment = a
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"case": app, "notes": notes, "documents": docs, "assessment": assessment})
}

func (h *Handler) LenderUpdateStatus(c *gin.Context) {
	_, lenderID, ok := h.requireLender(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	if _, err := h.svc.GetLenderCase(c.Request.Context(), lenderID, id); err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
			httpx.Forbidden(c, "This file is not in your pipeline.")
			return
		}
		httpx.Internal(c, "Could not load case.")
		return
	}
	h.updateStatus(c, lenderWorkingStatuses, "Lenders can mark in review or request more information. An admin records the top-level case outcome.")
}

func (h *Handler) LenderAddNote(c *gin.Context) {
	user, lenderID, ok := h.requireLender(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid case id.")
		return
	}
	if _, err := h.svc.GetLenderCase(c.Request.Context(), lenderID, id); err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
			httpx.Forbidden(c, "This file is not in your pipeline.")
			return
		}
		httpx.Internal(c, "Could not load case.")
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
	vis := body.Visibility
	if vis == "" || vis == "internal" {
		vis = "lender"
	}
	note, err := h.svc.AddNote(c.Request.Context(), user.ID, id, body.Body, vis)
	if err != nil {
		httpx.Internal(c, "Could not add note.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"note": note})
}
