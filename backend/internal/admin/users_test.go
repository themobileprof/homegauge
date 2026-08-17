package admin

import "testing"

func TestLooksLikeEmail(t *testing.T) {
	ok := []string{"admin@homegauge.local", "ada@bank.ng", "a.b@x.co"}
	for _, e := range ok {
		if !looksLikeEmail(e) {
			t.Fatalf("expected valid: %s", e)
		}
	}
	bad := []string{"", "nope", "a@", "@b.com", "a b@c.com"}
	for _, e := range bad {
		if looksLikeEmail(e) {
			t.Fatalf("expected invalid: %s", e)
		}
	}
}

func TestWouldDropLastAdmin(t *testing.T) {
	admin := &adminUser{Role: "ADMIN", Status: "active"}
	if !wouldDropLastAdmin(admin, "CUSTOMER", "active", false) {
		t.Fatal("demote should drop")
	}
	if !wouldDropLastAdmin(admin, "ADMIN", "disabled", false) {
		t.Fatal("disable should drop")
	}
	if !wouldDropLastAdmin(admin, "ADMIN", "active", true) {
		t.Fatal("delete should drop")
	}
	if wouldDropLastAdmin(admin, "ADMIN", "active", false) {
		t.Fatal("unchanged admin should not drop")
	}
	customer := &adminUser{Role: "CUSTOMER", Status: "active"}
	if wouldDropLastAdmin(customer, "ADVISOR", "disabled", true) {
		t.Fatal("non-admin should not trigger last-admin guard")
	}
}
