package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ConciergeDraft struct {
	Message     string   `json:"message"`
	Actions     []string `json:"actions"`
	Priority    string   `json:"priority"`
	Rationale   string   `json:"rationale,omitempty"`
	Provider    string   `json:"provider,omitempty"`
	Model       string   `json:"model,omitempty"`
	RawFallback string   `json:"raw_fallback,omitempty"`
}

type MultiReview struct {
	Primary   ConciergeDraft   `json:"primary"`
	Reviews   []ConciergeDraft `json:"reviews,omitempty"`
	Providers []string         `json:"providers"`
}

const conciergeSystem = `You are HomeGauge's mortgage concierge assistant for salaried applicants.
HomeGauge is NOT a bank and must never claim a loan is approved.
Return practical next steps for a human advisor reviewing a case.
Keep language plain and cautious.`

// DraftAdvisorReview asks configured models for advisor next-step suggestions.
func (c *Client) DraftAdvisorReview(ctx context.Context, caseBrief string) (*MultiReview, error) {
	if len(c.ConfiguredProviders()) == 0 {
		return nil, ErrNoProvider
	}
	req := CompletionRequest{
		System: conciergeSystem,
		User: fmt.Sprintf(`Given this mortgage enablement case, draft advisor guidance as JSON with keys:
message (string), actions (array of short snake_case action ids), priority (low|medium|high), rationale (short string).

Case brief:
%s`, caseBrief),
		JSON: true,
	}

	completions := c.CompleteAll(ctx, req)
	if len(completions) == 0 {
		// try primary once more for clearer error
		_, err := c.Complete(ctx, req)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("ai: no usable completions")
	}

	out := &MultiReview{Providers: make([]string, 0, len(completions))}
	for i, comp := range completions {
		draft := parseConciergeDraft(comp)
		out.Providers = append(out.Providers, string(comp.Provider))
		if i == 0 {
			out.Primary = draft
		} else {
			out.Reviews = append(out.Reviews, draft)
		}
	}
	return out, nil
}

func parseConciergeDraft(comp Completion) ConciergeDraft {
	raw := extractJSONObject(comp.Text)
	var d ConciergeDraft
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		d = ConciergeDraft{
			Message:     strings.TrimSpace(comp.Text),
			Actions:     []string{"review_case"},
			Priority:    "medium",
			RawFallback: comp.Text,
		}
	}
	if d.Message == "" {
		d.Message = "Review this customer case and confirm next document steps."
	}
	if len(d.Actions) == 0 {
		d.Actions = []string{"review_documents"}
	}
	if d.Priority == "" {
		d.Priority = "medium"
	}
	d.Provider = string(comp.Provider)
	d.Model = comp.Model
	return d
}
