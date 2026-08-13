package eligibility

import "testing"

func TestLikelyEligible(t *testing.T) {
	ev := Evaluate(Context{
		"monthly_income":   800000.0,
		"age":              34.0,
		"employment_type":  "salaried",
		"equity_pct":       20.0,
		"salary_months":    6.0,
	}, []Rule{
		{Field: "monthly_income", Operator: "gte", Value: 500000.0, Severity: "hard", MessageTemplate: "Income should be at least {value}."},
		{Field: "age", Operator: "lte", Value: 55.0, Severity: "hard"},
		{Field: "employment_type", Operator: "in", Value: []any{"salaried", "civil_servant"}, Severity: "hard"},
		{Field: "equity_pct", Operator: "gte", Value: 10.0, Severity: "hard"},
		{Field: "salary_months", Operator: "gte", Value: 6.0, Severity: "hard"},
	})
	if ev.Outcome != LikelyEligible {
		t.Fatalf("got %s", ev.Outcome)
	}
}

func TestHardFail(t *testing.T) {
	ev := Evaluate(Context{"monthly_income": 200000.0}, []Rule{
		{Field: "monthly_income", Operator: "gte", Value: 500000.0, Severity: "hard"},
	})
	if ev.Outcome != Unlikely {
		t.Fatalf("got %s", ev.Outcome)
	}
}

func TestMissingInfo(t *testing.T) {
	ev := Evaluate(Context{}, []Rule{
		{Field: "salary_months", Operator: "gte", Value: 6.0, Severity: "hard"},
	})
	if ev.Outcome != MoreInfoRequired {
		t.Fatalf("got %s", ev.Outcome)
	}
}
