package documents

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
	"github.com/homegauge/homegauge/backend/internal/platform/storage"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterCustomer(rg *gin.RouterGroup) {
	rg.GET("/documents/checklist", h.Checklist)
	rg.POST("/documents/upload", h.Upload)
	rg.GET("/documents/:id/download-url", h.DownloadURL)
}

func (h *Handler) RegisterPublicSigned(rg *gin.RouterGroup) {
	rg.GET("/documents/file", h.FileByToken)
}

func (h *Handler) RegisterStaff(rg *gin.RouterGroup) {
	rg.POST("/documents/:id/review", h.Review)
	rg.GET("/documents/:id/download-url", h.StaffDownloadURL)
}

func (h *Handler) Checklist(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	var productID *uuid.UUID
	if raw := c.Query("product_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.BadRequest(c, "Invalid product id.")
			return
		}
		productID = &id
	}
	items, appID, err := h.svc.Checklist(c.Request.Context(), user.ID, productID)
	if err != nil {
		httpx.Internal(c, "Could not load checklist.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"application_id": appID, "items": items})
}

func (h *Handler) Upload(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	appID, err := uuid.Parse(c.PostForm("application_id"))
	if err != nil {
		httpx.BadRequest(c, "application_id is required.")
		return
	}
	docType := strings.TrimSpace(c.PostForm("document_type_code"))
	if docType == "" {
		httpx.BadRequest(c, "document_type_code is required.")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpx.BadRequest(c, "file is required.")
		return
	}
	defer file.Close()

	mime := header.Header.Get("Content-Type")
	if mime == "" || mime == "application/octet-stream" {
		mime = sniffMIME(header.Filename)
	}
	doc, err := h.svc.Upload(c.Request.Context(), user.ID, appID, docType, mime, file)
	if errors.Is(err, ErrInvalidFile) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrApplicationNF) {
		httpx.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not upload document.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"document": doc})
}

func (h *Handler) DownloadURL(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid document id.")
		return
	}
	token, exp, err := h.svc.DownloadURL(c.Request.Context(), user.ID, id, false)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) {
		httpx.NotFound(c, "Document not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not create download link.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":        "/api/v1/documents/file?token=" + token,
		"expires_at": exp.Format(time.RFC3339),
	})
}

func (h *Handler) StaffDownloadURL(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid document id.")
		return
	}
	token, exp, err := h.svc.DownloadURL(c.Request.Context(), user.ID, id, true)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Document not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not create download link.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":        "/api/v1/documents/file?token=" + token,
		"expires_at": exp.Format(time.RFC3339),
	})
}

func (h *Handler) FileByToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		httpx.Unauthorized(c, "Missing token.")
		return
	}
	rc, mime, name, err := h.svc.OpenByToken(c.Request.Context(), token)
	if errors.Is(err, storage.ErrInvalidSignature) {
		httpx.Unauthorized(c, "Download link invalid or expired.")
		return
	}
	if err != nil {
		httpx.NotFound(c, "File not found.")
		return
	}
	defer rc.Close()
	c.Header("Content-Disposition", "attachment; filename="+name)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, -1, mime, rc, nil)
}

func (h *Handler) Review(c *gin.Context) {
	user := c.MustGet("auth_user").(auth.SessionUser)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid document id.")
		return
	}
	var body struct {
		Decision string `json:"decision" binding:"required"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "decision is required.")
		return
	}
	if err := h.svc.Review(c.Request.Context(), user.ID, id, body.Decision, body.Notes); err != nil {
		httpx.BadRequest(c, "Could not review document.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func sniffMIME(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
