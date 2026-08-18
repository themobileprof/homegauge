package applications

// Working statuses advisors may set while they handle the file.
// Terminal / top-level outcomes stay with admin.
var advisorWorkingStatuses = map[string]bool{
	"DOCUMENTS_PENDING":               true,
	"DOCUMENTS_UNDER_REVIEW":          true,
	"READY_FOR_SUBMISSION":            true,
	"SUBMITTED_TO_LENDER":             true,
	"LENDER_REVIEW":                   true,
	"ADDITIONAL_INFORMATION_REQUIRED": true,
}

// Lender portal users may only move files already submitted to their organisation.
var lenderWorkingStatuses = map[string]bool{
	"SUBMITTED_TO_LENDER":             true,
	"LENDER_REVIEW":                   true,
	"ADDITIONAL_INFORMATION_REQUIRED": true,
}

var allCaseStatuses = map[string]bool{
	"NEW":                             true,
	"ASSESSMENT_COMPLETED":            true,
	"DOCUMENTS_PENDING":               true,
	"DOCUMENTS_UNDER_REVIEW":          true,
	"READY_FOR_SUBMISSION":            true,
	"SUBMITTED_TO_LENDER":             true,
	"LENDER_REVIEW":                   true,
	"ADDITIONAL_INFORMATION_REQUIRED": true,
	"APPROVED":                        true,
	"REJECTED":                        true,
	"COMPLETED":                       true,
	"CANCELLED":                       true,
}

func advisorMaySetStatus(status string) bool {
	return advisorWorkingStatuses[status]
}

func lenderMaySetStatus(status string) bool {
	return lenderWorkingStatuses[status]
}

func adminMaySetStatus(status string) bool {
	return allCaseStatuses[status]
}
