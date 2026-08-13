package readiness

import "testing"

func TestComputeStrongSalary(t *testing.T) {
	s := Compute(Input{
		SalaryMonthsDetected: 6,
		SalaryVarianceOK:     true,
		DeclaredSalary:       900000,
		Deposit:              4000000,
		PropertyPrice:        35000000,
		MinEquityPct:         10,
		ITIPct:               32,
		YearsEmployed:        2,
		DocsReadyPct:         40,
		LikelyEligibleCount:  2,
		ProductCount:         3,
	})
	if s.Total < 70 {
		t.Fatalf("expected strong score, got %d (%s)", s.Total, s.Narrative)
	}
}

func TestWeakWithoutSalary(t *testing.T) {
	s := Compute(Input{ProductCount: 3})
	if s.Total > 20 {
		t.Fatalf("expected low score, got %d", s.Total)
	}
}
