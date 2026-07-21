package api

import (
	"fmt"
	"math"
	"testing"
)

// Balance is a money-display path: the live /user/me "wallet" field reports the
// balance scaled by 100 (regression fixed in 22bc0ef). These pin the scaling,
// the zero case, the fallback field names, and the "not present" signal so a
// future refactor can't silently show a 100x-wrong wallet before a real call.
func TestUserInfoBalance(t *testing.T) {
	cases := []struct {
		name    string
		info    UserInfo
		want    float64
		wantOK  bool
	}{
		{"wallet scaled by 100", UserInfo{"wallet": 9930.01}, 99.3001, true},
		{"wallet zero balance", UserInfo{"wallet": 0.0}, 0, true},
		{"wallet takes precedence over fallbacks", UserInfo{"wallet": 500.0, "balance": 999.0}, 5, true},
		{"fallback wallet_balance unscaled", UserInfo{"wallet_balance": 12.5}, 12.5, true},
		{"fallback balance unscaled", UserInfo{"balance": 7.0}, 7, true},
		{"fallback account_balance unscaled", UserInfo{"account_balance": 3.0}, 3, true},
		{"no balance field", UserInfo{"email": "x@y.z"}, 0, false},
		{"wallet wrong type is not a balance", UserInfo{"wallet": "9930.01"}, 0, false},
		{"empty", UserInfo{}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.info.Balance()
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if math.Abs(got-tc.want) > 1e-9 { // /100 scaling can drift by a ULP; exact == is a trap
				t.Errorf("balance = %v, want %v", got, tc.want)
			}
		})
	}
}

// The confirmation summary before a real, billable call prints the wallet as
// "$%.2f". A scaling regression would surface here as a 100x-wrong figure the
// user approves against, so pin the exact rendered string.
func TestBalanceConfirmationFormat(t *testing.T) {
	bal, ok := UserInfo{"wallet": 9930.01}.Balance()
	if !ok {
		t.Fatal("expected a balance")
	}
	if got := fmt.Sprintf("$%.2f", bal); got != "$99.30" {
		t.Errorf("formatted wallet = %q, want %q", got, "$99.30")
	}
}
