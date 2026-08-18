package applications

import "testing"

func TestAdvisorMaySetStatus(t *testing.T) {
	if !advisorMaySetStatus("DOCUMENTS_UNDER_REVIEW") {
		t.Fatal("advisor should set working status")
	}
	if advisorMaySetStatus("APPROVED") || advisorMaySetStatus("REJECTED") || advisorMaySetStatus("COMPLETED") || advisorMaySetStatus("CANCELLED") {
		t.Fatal("advisor should not set terminal status")
	}
	if !lenderMaySetStatus("LENDER_REVIEW") || !lenderMaySetStatus("ADDITIONAL_INFORMATION_REQUIRED") {
		t.Fatal("lender should update their pipeline")
	}
	if lenderMaySetStatus("APPROVED") || lenderMaySetStatus("READY_FOR_SUBMISSION") {
		t.Fatal("lender should not set top-level or pre-submit status")
	}
}

func TestAdminMaySetStatus(t *testing.T) {
	if !adminMaySetStatus("APPROVED") || !adminMaySetStatus("DOCUMENTS_PENDING") {
		t.Fatal("admin should set any known status")
	}
	if adminMaySetStatus("WEIRD") {
		t.Fatal("unknown status")
	}
}
