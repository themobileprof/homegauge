package calculator

import "testing"

func TestAffordabilityMREIFExample(t *testing.T) {
	// ₦30m loan @ 9.75% over 20 years — order-of-magnitude check
	res := Affordability(Input{
		PropertyPrice:      33_333_333,
		Deposit:            3_333_333,
		LoanAmount:         30_000_000,
		AnnualInterestRate: 9.75,
		TenorYears:         20,
		MonthlyIncome:      1_000_000,
		ExistingMonthlyDebt: 0,
	})
	if res.MonthlyRepayment < 250_000 || res.MonthlyRepayment > 320_000 {
		t.Fatalf("unexpected monthly repayment: %.2f", res.MonthlyRepayment)
	}
	if res.LoanToValuePct < 89 || res.LoanToValuePct > 91 {
		t.Fatalf("unexpected LTV: %.2f", res.LoanToValuePct)
	}
	if res.InstallmentToIncomePct < 25 || res.InstallmentToIncomePct > 35 {
		t.Fatalf("unexpected ITI: %.2f", res.InstallmentToIncomePct)
	}
}

func TestZeroInterest(t *testing.T) {
	res := Affordability(Input{LoanAmount: 1_200_000, AnnualInterestRate: 0, TenorYears: 1, MonthlyIncome: 500_000})
	if res.MonthlyRepayment != 100_000 {
		t.Fatalf("got %.2f", res.MonthlyRepayment)
	}
}
