package api

// UserInfo is the GET /user/me response: account profile, wallet balance,
// concurrency limits. Kept generic since the exact field set isn't
// documented and may grow; display code pulls out known fields defensively.
type UserInfo map[string]any

func (u UserInfo) str(key string) string {
	if v, ok := u[key].(string); ok {
		return v
	}
	return ""
}

func (u UserInfo) num(key string) (float64, bool) {
	v, ok := u[key].(float64)
	return v, ok
}

func (u UserInfo) Email() string { return u.str("email") }
func (u UserInfo) Name() string  { return u.str("name") }

// Balance returns the wallet balance if present under any of the field
// names Bolna has used across API versions. "wallet" (the live /user/me
// response) is confirmed to report the balance scaled by 100 — e.g. a raw
// value of 9930.01 for an actual balance of $99.30 — so it's divided back
// down here; the older/alternate fallback field names are unverified
// guesses and are returned as-is.
func (u UserInfo) Balance() (float64, bool) {
	if v, ok := u.num("wallet"); ok {
		return v / 100, true
	}
	for _, key := range []string{"wallet_balance", "balance", "account_balance"} {
		if v, ok := u.num(key); ok {
			return v, true
		}
	}
	return 0, false
}

// Concurrency returns the account's current/max concurrent call limits from
// the nested "concurrency": {"current": N, "max": M} object.
func (u UserInfo) Concurrency() (current, max int, ok bool) {
	nested, isMap := u["concurrency"].(map[string]any)
	if !isMap {
		return 0, 0, false
	}
	c, cOK := nested["current"].(float64)
	m, mOK := nested["max"].(float64)
	if !cOK || !mOK {
		return 0, 0, false
	}
	return int(c), int(m), true
}

func (c *Client) GetUserInfo() (UserInfo, error) {
	var info UserInfo
	err := c.do("/user/me", requestOptions{}, &info)
	return info, err
}
