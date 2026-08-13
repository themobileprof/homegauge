package mortgages

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/lenders", h.ListLenders)
	rg.GET("/mortgage-products", h.ListProducts)
	rg.GET("/mortgage-products/:id", h.GetProduct)
	rg.POST("/mortgage-products/compare", h.Compare)
}

func (h *Handler) ListLenders(c *gin.Context) {
	items, err := h.svc.ListLenders(c.Request.Context(), c.Query("country"))
	if err != nil {
		httpx.Internal(c, "Could not load lenders.")
		return
	}
	if items == nil {
		items = []Lender{}
	}
	c.JSON(http.StatusOK, gin.H{"lenders": items})
}

func (h *Handler) ListProducts(c *gin.Context) {
	items, err := h.svc.ListProducts(c.Request.Context(), c.Query("country"))
	if err != nil {
		httpx.Internal(c, "Could not load mortgage products.")
		return
	}
	if items == nil {
		items = []Product{}
	}
	c.JSON(http.StatusOK, gin.H{"products": items})
}

func (h *Handler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "Invalid product id.")
		return
	}
	p, err := h.svc.GetProduct(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(c, "Mortgage product not found.")
		return
	}
	if err != nil {
		httpx.Internal(c, "Could not load product.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

type compareRequest struct {
	ProductIDs []string `json:"product_ids" binding:"required,min=2,max=4"`
}

func (h *Handler) Compare(c *gin.Context) {
	var req compareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "Select 2 to 4 products to compare.")
		return
	}
	var products []Product
	for _, raw := range req.ProductIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.BadRequest(c, "Invalid product id.")
			return
		}
		p, err := h.svc.GetProduct(c.Request.Context(), id)
		if errors.Is(err, ErrNotFound) {
			httpx.NotFound(c, "One of the products was not found.")
			return
		}
		if err != nil {
			httpx.Internal(c, "Could not compare products.")
			return
		}
		products = append(products, *p)
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}
