package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DocumentExtraction is the only AI-shaped job we plan to use by default:
// unstructured bank statements / uploads. Eligibility, affordability, readiness,
// and advisor checklists stay programmatic.

type SalaryCredit struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
}

type SalaryExtraction struct {
	MonthsFound   int           `json:"months_found"`
	MedianCredit  float64       `json:"median_credit"`
	Credits       []SalaryCredit `json:"credits"`
	Gaps          []string      `json:"gaps"`
	Confidence    float64       `json:"confidence"`
	Notes         string        `json:"notes,omitempty"`
	Provider      string        `json:"provider,omitempty"`
	Model         string        `json:"model,omitempty"`
}

const extractSystem = `Extract recurring salary credits from bank statement text.
Return JSON only with keys: months_found (int), median_credit (number), credits (array of {date, amount, description}), gaps (string array), confidence (0-1), notes (string).
Do not invent transactions. If unclear, lower confidence and explain in notes.`

// ExtractSalaryPattern uses the documents-preferred model on statement text.
// Call only when OCR/text extraction of an uploaded statement is available.
func (c *Client) ExtractSalaryPattern(ctx context.Context, statementText string) (*SalaryExtraction, error) {
	req := CompletionRequest{
		System: extractSystem,
		User:   "Bank statement text:\n\n" + statementText,
		JSON:   true,
	}
	comp, err := c.Complete(ctx, JobDocuments, req)
	if err != nil {
		return nil, err
	}
	raw := extractJSONObject(comp.Text)
	var out SalaryExtraction
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("ai salary extract parse: %w", err)
	}
	out.Provider = string(comp.Provider)
	out.Model = comp.Model
	if out.Confidence == 0 && out.MonthsFound == 0 && len(out.Credits) == 0 {
		return nil, errors.New("ai salary extract: empty result")
	}
	return &out, nil
}

// ExplainNumerics is reserved for rare free-text explanation of already-computed
// calculator/eligibility numbers. Prefer showing the numbers themselves in UI.
func (c *Client) ExplainNumerics(ctx context.Context, computedSummary string) (string, error) {
	req := CompletionRequest{
		System: "Explain these already-calculated mortgage figures in plain language. Do not recalculate. Never say the applicant is approved.",
		User:   computedSummary,
		JSON:   false,
	}
	comp, err := c.Complete(ctx, JobNumerics, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(comp.Text), nil
}
