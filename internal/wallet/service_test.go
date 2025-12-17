package wallet

import "testing"

func TestValidateMoneyReq(t *testing.T) {
	if err := validateMoneyReq("w", "wallet", 1, "USD", "k"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := validateMoneyReq("", "wallet", 1, "USD", "k"); err == nil {
		t.Fatalf("expected error")
	}
}
