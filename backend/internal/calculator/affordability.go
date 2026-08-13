package calculator

import "math"

type Input struct {
	PropertyPrice       float64
	Deposit             float64
	LoanAmount          float64
	AnnualInterestRate  float64 // percent e.g. 9.75
	TenorYears          int
	MonthlyIncome       float64
	ExistingMonthlyDebt float64
	OtherMonthlyExpenses float64
}

type Result struct {
	LoanAmount              float64 `json:"loan_amount"`
	MonthlyRepayment        float64 `json:"monthly_repayment"`
	TotalRepayment          float64 `json:"total_repayment"`
	TotalInterest           float64 `json:"total_interest"`
	LoanToValuePct          float64 `json:"loan_to_value_pct"`
	DebtToIncomePct         float64 `json:"debt_to_income_pct"`
	RequiredEquity          float64 `json:"required_equity"`
	RequiredEquityPct       float64 `json:"required_equity_pct"`
	InstallmentToIncomePct  float64 `json:"installment_to_income_pct"`
	Disclaimer              string  `json:"disclaimer"`
}

// Affordability uses reducing-balance (amortizing) monthly payments.
func Affordability(in Input) Result {
	loan := in.LoanAmount
	if loan <= 0 && in.PropertyPrice > 0 {
		loan = math.Max(0, in.PropertyPrice-in.Deposit)
	}
	months := in.TenorYears * 12
	monthlyRate := (in.AnnualInterestRate / 100) / 12

	var payment float64
	if months <= 0 {
		payment = 0
	} else if monthlyRate == 0 {
		payment = loan / float64(months)
	} else {
		factor := math.Pow(1+monthlyRate, float64(months))
		payment = loan * monthlyRate * factor / (factor - 1)
	}

	total := payment * float64(months)
	ltv := 0.0
	if in.PropertyPrice > 0 {
		ltv = (loan / in.PropertyPrice) * 100
	}
	requiredEquity := math.Max(0, in.PropertyPrice-loan)
	requiredEquityPct := 0.0
	if in.PropertyPrice > 0 {
		requiredEquityPct = (requiredEquity / in.PropertyPrice) * 100
	}
	dti := 0.0
	iti := 0.0
	if in.MonthlyIncome > 0 {
		dti = ((payment + in.ExistingMonthlyDebt) / in.MonthlyIncome) * 100
		iti = (payment / in.MonthlyIncome) * 100
	}

	return Result{
		LoanAmount:             round2(loan),
		MonthlyRepayment:       round2(payment),
		TotalRepayment:         round2(total),
		TotalInterest:          round2(total - loan),
		LoanToValuePct:         round2(ltv),
		DebtToIncomePct:        round2(dti),
		RequiredEquity:         round2(requiredEquity),
		RequiredEquityPct:      round2(requiredEquityPct),
		InstallmentToIncomePct: round2(iti),
		Disclaimer:             "These figures are estimates only. They are not a loan offer or bank approval.",
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
