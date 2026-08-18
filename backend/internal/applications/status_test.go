package applications

import "testing"

func TestAdvisorMaySetStatus(t *testing.T) {
	if !advisorMaySetStatus("DOCUMENTS_UNDER_REVIEW") {
		t.Fatal("advisor should set working status")
	}
	if advisorMaySetStatus("APPROVED") || advisorMaySetStatus("REJECTED") || advisorMaySetStatus("COMPLETED") || advisorMaySetStatus("CANCELLED") {
		t.Fatal("advisor should not set terminal status")
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
