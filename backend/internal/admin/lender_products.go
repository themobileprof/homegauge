package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

// lenderProductWrite is the body lenders submit. Lender and verification are server-controlled.
type lenderProductWrite struct {
	CountryCode      string   `json:"country_code"`
	Name             string   `json:"name" binding:"required,min=2"`
	Description      string   `json:"description"`
	MortgageType     string   `json:"mortgage_type" binding:"required"`
	MinLoanAmount    *float64 `json:"min_loan_amount"`
	MaxLoanAmount    *float64 `json:"max_loan_amount"`
	MinIncome        *float64 `json:"min_income"`
	MaxAge           *int     `json:"max_age"`
	MaxTenorYears    *int     `json:"max_tenor_years"`
	MinEquityPct     *float64 `json:"min_equity_pct"`
	InterestRate     *float64 `json:"interest_rate"`
	InterestRateMin  *float64 `json:"interest_rate_min"`
	InterestRateMax  *float64 `json:"interest_rate_max"`
	InterestRateType string   `json:"interest_rate_type"`
	ProcessingFee    *float64 `json:"processing_fee"`
	ValuationFee     *float64 `json:"valuation_fee"`
	LegalFee         *float64 `json:"legal_fee"`
	Status           string   `json:"status"`
	Source           string   `json:"source"`
	SourceURL        string   `json:"source_url"`
	SyncRules        *bool    `json:"sync_rules"`
}

func (h *Handler) RegisterLender(rg *gin.RouterGroup) {
	rg.GET("/products", h.LenderListProducts)
	rg.GET("/products/:id", h.LenderGetProduct)
	rg.POST("/products", h.LenderCreateProduct)
	rg.PATCH("/products/:id", h.LenderUpdateProduct)
}

func (h *Handler) LenderListProducts(c *gin.Context) {
	_, lenderID, ok := h.requireLenderSession(c)
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(c.Request.Context(),
		productSelect+` WHERE p.deleted_at IS NULL AND p.lender_id = $1::uuid ORDER BY p.updated_at DESC`,
		lenderID.String(),
	)
	if err != nil {
		httpx.Internal(c, "Could not load products.")
		return
	}
	defer rows.Close()
	out := []adminProduct{}
	for rows.Next() {
		p, err := scanAdminProduct(rows)
		if err != nil {
			httpx.Internal(c, "Could not load products.")
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"products": out})
}

func (h *Handler) LenderGetProduct(c *gin.Context) {
	_, lenderID, ok := h.requireLenderSession(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown product.")
		return
	}
	p, okOwned := h.getLenderOwnedProduct(c, lenderID, id)
	if !okOwned {
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

func (h *Handler) LenderCreateProduct(c *gin.Context) {
	_, lenderID, ok := h.requireLenderSession(c)
	if !ok {
		return
	}
	org, err := h.getLender(c.Request.Context(), lenderID.String())
	if err != nil {
		httpx.Internal(c, "Could not load your lender profile.")
		return
	}
	var in lenderProductWrite
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Name and mortgage type are required.")
		return
	}
	write, msg := lenderWriteToProduct(in, lenderID.String(), org.CountryCode)
	if msg != "" {
		httpx.BadRequest(c, msg)
		return
	}
	write.VerificationStatus = "needs_verification"

	var id string
	err = h.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO mortgage_products (
			lender_id, country_code, name, description, mortgage_type,
			min_loan_amount, max_loan_amount, min_income, max_age, max_tenor_years, min_equity_pct,
			interest_rate, interest_rate_min, interest_rate_max, interest_rate_type, processing_fee, valuation_fee, legal_fee,
			status, source, source_url, verification_status, last_verified_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18,
			$19, NULLIF($20,''), NULLIF($21,''), 'needs_verification'::verification_status, NULL
		) RETURNING id::text
	`, write.LenderID, write.CountryCode, strings.TrimSpace(write.Name), strings.TrimSpace(write.Description), write.MortgageType,
		write.MinLoanAmount, write.MaxLoanAmount, write.MinIncome, write.MaxAge, write.MaxTenorYears, write.MinEquityPct,
		write.InterestRate, write.InterestRateMin, write.InterestRateMax, defaultRateType(write.InterestRateType), write.ProcessingFee, write.ValuationFee, write.LegalFee,
		defaultProductStatus(write.Status), strings.TrimSpace(write.Source), strings.TrimSpace(write.SourceURL),
	).Scan(&id)
	if err != nil {
		httpx.Internal(c, "Could not create product.")
		return
	}
	if in.SyncRules == nil || *in.SyncRules {
		if err := h.syncDerivedRules(c.Request.Context(), id, write, true); err != nil {
			httpx.Internal(c, "Product saved but eligibility rules could not be updated.")
			return
		}
	}
	if err := h.ensureDefaultDocuments(c.Request.Context(), id); err != nil {
		httpx.Internal(c, "Product saved but the default document list could not be attached.")
		return
	}
	p, err := h.getProduct(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Product created but could not be loaded.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": p})
}

func (h *Handler) LenderUpdateProduct(c *gin.Context) {
	_, lenderID, ok := h.requireLenderSession(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.BadRequest(c, "Unknown product.")
		return
	}
	if _, okOwned := h.getLenderOwnedProduct(c, lenderID, id); !okOwned {
		return
	}
	org, err := h.getLender(c.Request.Context(), lenderID.String())
	if err != nil {
		httpx.Internal(c, "Could not load your lender profile.")
		return
	}
	var in lenderProductWrite
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "Name and mortgage type are required.")
		return
	}
	write, msg := lenderWriteToProduct(in, lenderID.String(), org.CountryCode)
	if msg != "" {
		httpx.BadRequest(c, msg)
		return
	}
	// Lender edits always need re-verification; they cannot self-verify.
	write.VerificationStatus = "needs_verification"

	_, err = h.db.ExecContext(c.Request.Context(), `
		UPDATE mortgage_products SET
			lender_id=$2::uuid, country_code=$3, name=$4, description=$5, mortgage_type=$6,
			min_loan_amount=$7, max_loan_amount=$8, min_income=$9, max_age=$10, max_tenor_years=$11, min_equity_pct=$12,
			interest_rate=$13, interest_rate_min=$14, interest_rate_max=$15, interest_rate_type=$16, processing_fee=$17, valuation_fee=$18, legal_fee=$19,
			status=$20, source=NULLIF($21,''), source_url=NULLIF($22,''),
			verification_status='needs_verification'::verification_status,
			updated_at=NOW()
		WHERE id=$1::uuid AND lender_id=$2::uuid AND deleted_at IS NULL
	`, id, write.LenderID, write.CountryCode, strings.TrimSpace(write.Name), strings.TrimSpace(write.Description), write.MortgageType,
		write.MinLoanAmount, write.MaxLoanAmount, write.MinIncome, write.MaxAge, write.MaxTenorYears, write.MinEquityPct,
		write.InterestRate, write.InterestRateMin, write.InterestRateMax, defaultRateType(write.InterestRateType), write.ProcessingFee, write.ValuationFee, write.LegalFee,
		defaultProductStatus(write.Status), strings.TrimSpace(write.Source), strings.TrimSpace(write.SourceURL),
	)
	if err != nil {
		httpx.Internal(c, "Could not update product.")
		return
	}
	if in.SyncRules == nil || *in.SyncRules {
		if err := h.syncDerivedRules(c.Request.Context(), id, write, false); err != nil {
			httpx.Internal(c, "Product saved but eligibility rules could not be updated.")
			return
		}
	}
	p, err := h.getProduct(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, "Could not load updated product.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

func (h *Handler) requireLenderSession(c *gin.Context) (auth.SessionUser, uuid.UUID, bool) {
	raw, exists := c.Get("auth_user")
	if !exists {
		httpx.Unauthorized(c, "Sign in required.")
		return auth.SessionUser{}, uuid.Nil, false
	}
	user, ok := raw.(auth.SessionUser)
	if !ok {
		httpx.Unauthorized(c, "Sign in required.")
		return auth.SessionUser{}, uuid.Nil, false
	}
	if user.LenderID == nil {
		httpx.Forbidden(c, "Your account is not linked to a lender.")
		return user, uuid.Nil, false
	}
	return user, *user.LenderID, true
}

func (h *Handler) getLenderOwnedProduct(c *gin.Context, lenderID uuid.UUID, productID string) (*adminProduct, bool) {
	p, err := h.getProduct(c.Request.Context(), productID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(c, "Product not found.")
		return nil, false
	}
	if err != nil {
		httpx.Internal(c, "Could not load product.")
		return nil, false
	}
	if p.LenderID != lenderID.String() {
		httpx.NotFound(c, "Product not found.")
		return nil, false
	}
	return p, true
}

func lenderWriteToProduct(in lenderProductWrite, lenderID, orgCountry string) (productWrite, string) {
	country := strings.ToUpper(strings.TrimSpace(in.CountryCode))
	if country == "" {
		country = strings.ToUpper(orgCountry)
	}
	if !strings.EqualFold(country, orgCountry) {
		return productWrite{}, "Products must stay in your lender’s country."
	}
	write := productWrite{
		LenderID:           lenderID,
		CountryCode:        country,
		Name:               in.Name,
		Description:        in.Description,
		MortgageType:       in.MortgageType,
		MinLoanAmount:      in.MinLoanAmount,
		MaxLoanAmount:      in.MaxLoanAmount,
		MinIncome:          in.MinIncome,
		MaxAge:             in.MaxAge,
		MaxTenorYears:      in.MaxTenorYears,
		MinEquityPct:       in.MinEquityPct,
		InterestRate:       in.InterestRate,
		InterestRateMin:    in.InterestRateMin,
		InterestRateMax:    in.InterestRateMax,
		InterestRateType:   in.InterestRateType,
		ProcessingFee:      in.ProcessingFee,
		ValuationFee:       in.ValuationFee,
		LegalFee:           in.LegalFee,
		Status:             in.Status,
		Source:             in.Source,
		SourceURL:          in.SourceURL,
		VerificationStatus: "needs_verification",
		SyncRules:          in.SyncRules,
	}
	if msg := validateProductWrite(write); msg != "" {
		return productWrite{}, msg
	}
	return write, ""
}
