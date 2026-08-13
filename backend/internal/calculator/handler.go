package calculator

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/affordability", h.Affordability)
}

type affordabilityRequest struct {
	PropertyPrice        float64 `json:"property_price"`
	Deposit              float64 `json:"deposit"`
	LoanAmount           float64 `json:"loan_amount"`
	AnnualInterestRate   float64 `json:"interest_rate" binding:"required"`
	TenorYears           int     `json:"tenor_years" binding:"required,min=1,max=35"`
	MonthlyIncome        float64 `json:"monthly_income"`
	ExistingMonthlyDebt  float64 `json:"existing_monthly_debt"`
	OtherMonthlyExpenses float64 `json:"other_monthly_expenses"`
}

func (h *Handler) Affordability(c *gin.Context) {
	var req affordabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "Please provide interest rate and tenor, plus property/loan details.")
		return
	}
	if req.LoanAmount <= 0 && req.PropertyPrice <= 0 {
		httpx.BadRequest(c, "Enter a property price or loan amount.")
		return
	}
	res := Affordability(Input{
		PropertyPrice:        req.PropertyPrice,
		Deposit:              req.Deposit,
		LoanAmount:           req.LoanAmount,
		AnnualInterestRate:   req.AnnualInterestRate,
		TenorYears:           req.TenorYears,
		MonthlyIncome:        req.MonthlyIncome,
		ExistingMonthlyDebt:  req.ExistingMonthlyDebt,
		OtherMonthlyExpenses: req.OtherMonthlyExpenses,
	})
	c.JSON(http.StatusOK, res)
}
