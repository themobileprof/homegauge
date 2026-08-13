package readiness

import "fmt"

type Input struct {
	SalaryMonthsDetected int
	SalaryVarianceOK     bool
	DeclaredSalary       float64
	Deposit              float64
	PropertyPrice        float64
	MinEquityPct         float64 // typical / best-fit requirement
	ITIPct               float64
	YearsEmployed        float64
	DocsReadyPct         float64 // 0-100
	LikelyEligibleCount  int
	ProductCount         int
}

type Component struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Score   int    `json:"score"`
	Max     int    `json:"max"`
	Note    string `json:"note"`
}

type Score struct {
	Total      int         `json:"total"`
	Components []Component `json:"components"`
	Narrative  string      `json:"narrative"`
}

// Weights (admin-configurable later): salary 30, deposit 20, debt/ITI 20, docs 15, employment 10, coverage 5
func Compute(in Input) Score {
	salary := clamp(0, 30, salaryScore(in))
	deposit := clamp(0, 20, depositScore(in))
	debt := clamp(0, 20, debtScore(in.ITIPct))
	docs := clamp(0, 15, int(in.DocsReadyPct*15/100))
	emp := clamp(0, 10, employmentScore(in.YearsEmployed))
	cov := clamp(0, 5, coverageScore(in.LikelyEligibleCount, in.ProductCount))

	components := []Component{
		{Key: "salary_pattern", Label: "Income stability (salary account)", Score: salary, Max: 30, Note: salaryNote(in)},
		{Key: "deposit_readiness", Label: "Deposit readiness", Score: deposit, Max: 20, Note: depositNote(in)},
		{Key: "debt_iti", Label: "Debt burden", Score: debt, Max: 20, Note: debtNote(in.ITIPct)},
		{Key: "documentation", Label: "Documentation readiness", Score: docs, Max: 15, Note: "Based on how complete your checklist is so far."},
		{Key: "employment_tenure", Label: "Employment history", Score: emp, Max: 10, Note: employmentNote(in.YearsEmployed)},
		{Key: "eligibility_coverage", Label: "Eligibility coverage", Score: cov, Max: 5, Note: coverageNote(in.LikelyEligibleCount, in.ProductCount)},
	}

	total := salary + deposit + debt + docs + emp + cov
	return Score{
		Total:      total,
		Components: components,
		Narrative:  narrative(components, total),
	}
}

func salaryScore(in Input) int {
	if in.SalaryMonthsDetected >= 6 && in.SalaryVarianceOK {
		return 30
	}
	if in.SalaryMonthsDetected >= 6 {
		return 22
	}
	if in.SalaryMonthsDetected >= 3 {
		return 12
	}
	if in.SalaryMonthsDetected > 0 {
		return 6
	}
	// Self-reported only — partial credit until statement analysis exists
	if in.DeclaredSalary > 0 {
		return 10
	}
	return 0
}

func depositScore(in Input) int {
	if in.PropertyPrice <= 0 {
		if in.Deposit > 0 {
			return 10
		}
		return 0
	}
	pct := (in.Deposit / in.PropertyPrice) * 100
	need := in.MinEquityPct
	if need <= 0 {
		need = 10
	}
	if pct >= need+10 {
		return 20
	}
	if pct >= need {
		return 16
	}
	if pct >= need*0.7 {
		return 10
	}
	if pct > 0 {
		return 5
	}
	return 0
}

func debtScore(iti float64) int {
	if iti <= 0 {
		return 12
	}
	if iti <= 30 {
		return 20
	}
	if iti <= 35 {
		return 16
	}
	if iti <= 40 {
		return 10
	}
	if iti <= 50 {
		return 5
	}
	return 0
}

func employmentScore(years float64) int {
	if years >= 2 {
		return 10
	}
	if years >= 1 {
		return 8
	}
	if years >= 0.5 {
		return 6
	}
	if years > 0 {
		return 3
	}
	return 0
}

func coverageScore(likely, total int) int {
	if total <= 0 {
		return 0
	}
	if likely >= 2 {
		return 5
	}
	if likely == 1 {
		return 3
	}
	return 1
}

func salaryNote(in Input) string {
	if in.SalaryMonthsDetected >= 6 && in.SalaryVarianceOK {
		return "Your salary credits look steady across about 6 months."
	}
	if in.DeclaredSalary > 0 && in.SalaryMonthsDetected == 0 {
		return "You declared a salary; upload 6 months of statements so we can confirm the pattern."
	}
	return "Lenders usually want clear salary credits for about 6 months on one account."
}

func depositNote(in Input) string {
	if in.PropertyPrice <= 0 {
		return "Add a property price to see how far your deposit goes."
	}
	pct := (in.Deposit / in.PropertyPrice) * 100
	return fmt.Sprintf("Your deposit is about %.0f%% of the property price.", pct)
}

func debtNote(iti float64) string {
	if iti <= 0 {
		return "We will refine this once we estimate your monthly mortgage payment."
	}
	return fmt.Sprintf("Estimated mortgage payment is about %.0f%% of your monthly income.", iti)
}

func employmentNote(years float64) string {
	if years >= 0.5 {
		return "Many products ask for at least 6 months with your current employer."
	}
	return "A longer time with your current employer usually helps."
}

func coverageNote(likely, total int) string {
	if likely > 0 {
		return fmt.Sprintf("You appear to meet stated criteria for %d of %d products reviewed.", likely, total)
	}
	return "No strong product matches yet based on the information provided."
}

func narrative(components []Component, total int) string {
	var weak, strong string
	minRatio, maxRatio := 2.0, -1.0
	for _, c := range components {
		ratio := float64(c.Score) / float64(c.Max)
		if ratio < minRatio {
			minRatio = ratio
			weak = c.Label
		}
		if ratio > maxRatio {
			maxRatio = ratio
			strong = c.Label
		}
	}
	return fmt.Sprintf(
		"Mortgage readiness %d/100. Strongest area: %s. Watch-out: %s. This is an educational score — not a bank credit score or approval.",
		total, strong, weak,
	)
}

func clamp(min, max, v int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
