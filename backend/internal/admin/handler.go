package admin

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/ai"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct {
	db *sql.DB
	ai *ai.Client
}

func NewHandler(db *sql.DB, aiClient *ai.Client) *Handler {
	return &Handler{db: db, ai: aiClient}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/overview", h.Overview)
	rg.GET("/users", h.ListUsers)
	rg.POST("/users", h.CreateUser)
	rg.PATCH("/users/:id", h.UpdateUser)
	rg.DELETE("/users/:id", h.DeleteUser)
	rg.GET("/lenders", h.ListLenders)
	rg.POST("/lenders", h.CreateLender)
	rg.GET("/products", h.ListProducts)
	rg.POST("/products", h.CreateProduct)
	rg.PATCH("/products/:id", h.UpdateProduct)
	rg.DELETE("/products/:id", h.DeleteProduct)
}

func (h *Handler) Overview(c *gin.Context) {
	counts := map[string]int{}
	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT role::text, COUNT(*) FROM users WHERE deleted_at IS NULL GROUP BY role
	`)
	if err != nil {
		httpx.Internal(c, "Could not load admin overview.")
		return
	}
	defer rows.Close()
	totalUsers := 0
	for rows.Next() {
		var role string
		var n int
		if err := rows.Scan(&role, &n); err != nil {
			httpx.Internal(c, "Could not load admin overview.")
			return
		}
		counts[role] = n
		totalUsers += n
	}

	var products, lenders, openCases, unassigned, readyForApproval int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM mortgage_products WHERE deleted_at IS NULL AND status='active'`).Scan(&products)
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM lenders WHERE deleted_at IS NULL AND status='active'`).Scan(&lenders)
	_ = h.db.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM mortgage_applications WHERE status NOT IN ('CANCELLED','COMPLETED','REJECTED')
	`).Scan(&openCases)
	_ = h.db.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM mortgage_applications
		WHERE assigned_advisor_id IS NULL AND status NOT IN ('CANCELLED','COMPLETED','REJECTED')
	`).Scan(&unassigned)
	_ = h.db.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM mortgage_applications WHERE status = 'READY_FOR_SUBMISSION'
	`).Scan(&readyForApproval)

	routing := map[string]string{}
	configured := []ai.ProviderName{}
	if h.ai != nil {
		routing = h.ai.JobRouting()
		configured = h.ai.ConfiguredProviders()
	}

	c.JSON(http.StatusOK, gin.H{
		"users_total":     totalUsers,
		"users_by_role":   counts,
		"active_products": products,
		"active_lenders":  lenders,
		"open_cases":           openCases,
		"unassigned_cases":     unassigned,
		"ready_for_approval":   readyForApproval,
		"ai": gin.H{
			"configured": configured,
			"routing":    routing,
		},
	})
}
