package eligibility

import (
	"encoding/json"
	"strings"
)

type Outcome string

const (
	LikelyEligible     Outcome = "likely_eligible"
	PotentiallyEligible Outcome = "potentially_eligible"
	MayRequireReview   Outcome = "may_require_review"
	Unlikely           Outcome = "unlikely"
	MoreInfoRequired   Outcome = "more_info_required"
)

type Rule struct {
	Field           string
	Operator        string
	Value           any
	Severity        string // hard | soft
	MessageTemplate string
}

type Context map[string]any

type RuleResult struct {
	Field   string `json:"field"`
	Passed  bool   `json:"passed"`
	Severity string `json:"severity"`
	Message string `json:"message"`
}

type Evaluation struct {
	Outcome Outcome      `json:"outcome"`
	Results []RuleResult `json:"results"`
	Summary string       `json:"summary"`
}

func Evaluate(ctx Context, rules []Rule) Evaluation {
	var results []RuleResult
	hardFail, softFail, missing := 0, 0, 0

	for _, r := range rules {
		val, ok := ctx[r.Field]
		if !ok || val == nil || val == "" {
			missing++
			results = append(results, RuleResult{
				Field: r.Field, Passed: false, Severity: r.Severity,
				Message: "More information is needed for: " + humanField(r.Field),
			})
			continue
		}
		passed := match(val, r.Operator, r.Value)
		msg := renderMessage(r.MessageTemplate, r.Value)
		if msg == "" {
			if passed {
				msg = humanField(r.Field) + " looks acceptable for this product."
			} else {
				msg = humanField(r.Field) + " does not meet this product’s stated criteria."
			}
		}
		results = append(results, RuleResult{Field: r.Field, Passed: passed, Severity: r.Severity, Message: msg})
		if !passed {
			if r.Severity == "soft" {
				softFail++
			} else {
				hardFail++
			}
		}
	}

	var outcome Outcome
	switch {
	case missing > 0 && hardFail == 0:
		outcome = MoreInfoRequired
	case hardFail > 0:
		outcome = Unlikely
	case softFail > 0:
		outcome = PotentiallyEligible
	case softFail == 0 && hardFail == 0 && missing == 0:
		outcome = LikelyEligible
	default:
		outcome = MayRequireReview
	}

	return Evaluation{
		Outcome: outcome,
		Results: results,
		Summary: outcomeSummary(outcome),
	}
}

func outcomeSummary(o Outcome) string {
	switch o {
	case LikelyEligible:
		return "Based on the information provided, you appear to meet the stated eligibility criteria. This is not a bank approval."
	case PotentiallyEligible:
		return "You may be a fit, but some criteria need a closer look. This is not a bank approval."
	case MayRequireReview:
		return "A human advisor should review your case before any next step with a lender."
	case Unlikely:
		return "Based on the information provided, you are unlikely to meet this product’s stated criteria."
	default:
		return "More information is required before we can assess this product."
	}
}

func match(actual any, op string, expected any) bool {
	switch op {
	case "gte":
		return asFloat(actual) >= asFloat(expected)
	case "lte":
		return asFloat(actual) <= asFloat(expected)
	case "eq":
		return stringify(actual) == stringify(expected)
	case "neq":
		return stringify(actual) != stringify(expected)
	case "in":
		for _, item := range asSlice(expected) {
			if stringify(actual) == stringify(item) {
				return true
			}
		}
		return false
	case "between":
		arr := asSlice(expected)
		if len(arr) < 2 {
			return false
		}
		v := asFloat(actual)
		return v >= asFloat(arr[0]) && v <= asFloat(arr[1])
	default:
		return false
	}
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		var f float64
		_ = json.Unmarshal([]byte(t), &f)
		return f
	default:
		b, _ := json.Marshal(v)
		var f float64
		_ = json.Unmarshal(b, &f)
		return f
	}
}

func asSlice(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	default:
		b, _ := json.Marshal(v)
		var arr []any
		_ = json.Unmarshal(b, &arr)
		return arr
	}
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(v)
		return strings.Trim(string(b), `"`)
	}
}

func renderMessage(tmpl string, value any) string {
	if tmpl == "" {
		return ""
	}
	return strings.ReplaceAll(tmpl, "{value}", stringify(value))
}

func humanField(f string) string {
	return strings.ReplaceAll(f, "_", " ")
}
