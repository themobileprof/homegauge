package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeEmail(t *testing.T) {
	got := normalizeEmail("  Ada@HomeGauge.NG ")
	if got != "ada@homegauge.ng" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeDecodeSession(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	raw := encodeSession(SessionUser{ID: id, Email: "a@b.com", Role: RoleCustomer})
	su, err := decodeSession(raw)
	if err != nil {
		t.Fatal(err)
	}
	if su.Email != "a@b.com" || su.Role != RoleCustomer || su.ID != id {
		t.Fatalf("unexpected session %+v", su)
	}
}

func TestHashTokenStable(t *testing.T) {
	a := hashToken("abc")
	b := hashToken("abc")
	if a != b || len(a) != 64 {
		t.Fatalf("hash mismatch")
	}
}
