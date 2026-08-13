package countries

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/countries", h.List)
	rg.GET("/countries/:code", h.Get)
}

func (h *Handler) List(c *gin.Context) {
	includeComing := c.Query("include") == "coming_soon" || c.Query("include") == "all"
	items, err := h.svc.List(c.Request.Context(), includeComing)
	if err != nil {
		httpx.Internal(c, "Could not load countries.")
		return
	}
	if items == nil {
		items = []Country{}
	}
	def, _ := h.svc.DefaultCode(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"countries": items, "default_country_code": def})
}

func (h *Handler) Get(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	item, err := h.svc.Get(c.Request.Context(), code)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Country not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load country.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"country": item})
}
